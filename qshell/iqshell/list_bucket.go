package iqshell

import (
	"bufio"
	"context"
	"fmt"
	"github.com/astaxie/beego/logs"
	"os"
	"os/signal"
	"strings"
	"time"
)

func filterByPuttime(putTime, startDate, endDate time.Time) bool {
	switch {
	case startDate.IsZero() && endDate.IsZero():
		return true
	case !startDate.IsZero() && endDate.IsZero() && putTime.After(startDate):
		return true
	case !endDate.IsZero() && startDate.IsZero() && putTime.Before(endDate):
		return true
	case putTime.After(startDate) && putTime.Before(endDate):
		return true
	default:
		return false
	}

}

func filterBySuffixes(key string, suffixes []string) bool {
	hasSuffix := false
	if len(suffixes) == 0 {
		hasSuffix = true
	}
	for _, s := range suffixes {
		if strings.HasSuffix(key, s) {
			hasSuffix = true
			break
		}
	}
	if hasSuffix {
		return true
	} else {
		return false
	}
}

func errorWarning(marker string, err error) {
	fmt.Fprintf(os.Stderr, "marker: %s\n", marker)
	fmt.Fprintf(os.Stderr, "listbucket Error: %v\n", err)
}

/*
*@param bucket
*@param prefix
*@param marker
*@param listResultFile
*@return listError
 */
func (m *BucketManager) ListFiles(bucket, prefix, marker, listResultFile string) (retErr error) {
	return m.ListBucket2(bucket, prefix, marker, listResultFile, "", time.Time{}, time.Time{}, nil, 20, false, false)
}

func (m *BucketManager) ListBucket2(bucket, prefix, marker, listResultFile, delimiter string, startDate, endDate time.Time, suffixes []string, maxRetry int, appendMode bool, readable bool) (retErr error) {
	lastMarker := marker

	defer func(lastMarker string) {
		if lastMarker != "" {
			fmt.Fprintf(os.Stderr, "Marker: %s\n", lastMarker)
		}
	}(lastMarker)

	/*
	Context：
	  主要的用处如果用一句话来说，是在于控制goroutine的生命周期。
	  当一个计算任务被goroutine承接了之后，由于某种原因（超时，或者强制退出）我们希望中止这个goroutine，或与其相关的其他goroutine的计算任务，那么就用得到这个Context了
	  所有的goroutine都需要内置处理这个听声器结束信号的逻辑（ctx->Done()），一旦满足(select选中)则对当前goroutine进行一定处理
	Context interface定义
	type Context interface {
	    // Done returns a channel(<-chan struct{}) that is closed when this Context is canceled
	    // or times out.
	    Done() <-chan struct{}
		// 该函数返回一个channel。当times out或者调用cancel方法时，该channel将会close掉

		// chan T          可以接收和发送类型为 T 的数据
		// chan<- float64  只可以用来发送 float64 类型的数据
		// <-chan int      只可以用来接收 int 类型的数据

	    // Err indicates why this context was canceled, after the Done channel
	    // is closed.
	    Err() error
		// 返回一个错误。该context为什么被取消掉

	    // Deadline returns the time when this Context will be canceled, if any.
	    Deadline() (deadline time.Time, ok bool)
		// 返回截止时间和ok

	    // Value returns the value associated with key or nil if none.
	    Value(key interface{}) interface{}
		// 返回值
	}

	导出方法
	func Background() Context
	func TODO() Context
	// Context是一个接口，想要使用就得实现其方法。在context包内部已经为我们实现好了两个空的Context，可以通过调用以上2个方法获取。一般的将它们作为Context的根，往下派生

	满足各种需求的预设上下文，包括超时/互相传递数据等...
	func WithCancel(parent Context) (ctx Context, cancel CancelFunc)  ==> cancel()时，<-ctx.Done()将会触发
	func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc)  ==> WithDeadline 和 WithTimeout 是相似的，WithDeadline 是设置具体的 deadline 时间，到达 deadline 的时候，后代 goroutine 退出
	func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) ==> WithTimeout 简单粗暴，直接 return WithDeadline(parent, time.Now().Add(timeout))
	func WithValue(parent Context, key, val interface{}) Context   ==> 互相传递数据, key不应设置成为普通String或Int，为防止不同中间件对key的覆盖。最好是每个中间件自定义key类型，而且获取Value的逻辑尽量抽象到一个函数。避免各种key的冲突问题
	 */

	sigChan := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(sigChan, os.Interrupt)

	go func() {
		// 捕捉Ctrl-C, 退出下面列举的循环
		<-sigChan
		cancel()
		maxRetry = 0

		fmt.Printf("\nMarker: %s\n", lastMarker)
		os.Exit(1)
	}()

	var listResultFh *os.File

	if listResultFile == "" {
		listResultFh = os.Stdout
	} else {
		var openErr error
		var mode int

		if appendMode {
			mode = os.O_APPEND | os.O_RDWR
		} else {
			mode = os.O_CREATE | os.O_RDWR | os.O_TRUNC
		}
		listResultFh, openErr = os.OpenFile(listResultFile, mode, 0666)
		if openErr != nil {
			retErr = openErr
			logs.Error("Failed to open list result file `%s`", listResultFile)
			return
		}
		defer listResultFh.Close()
	}

	bWriter := bufio.NewWriter(listResultFh)

	notfilterTime := startDate.IsZero() && endDate.IsZero()
	notfilterSuffix := len(suffixes) == 0

	var c int
	for {
		if maxRetry >= 0 && c >= maxRetry {
			break
		}
		entries, lErr := m.ListBucketContext(ctx, bucket, prefix, delimiter, marker)

		if entries == nil && lErr == nil {
			// no data
			if lastMarker == "" {
				break
			} else {
				fmt.Fprintf(os.Stderr, "meet empty body when list not completed\n")
				continue
			}
		}
		if lErr != nil {
			retErr = lErr
			errorWarning(lastMarker, retErr)
			if maxRetry > 0 {
				c++
			}
			time.Sleep(1)
			continue
		}
		var fsizeValue interface{}

		for listItem := range entries {
			if listItem.Marker != lastMarker {
				lastMarker = listItem.Marker
			}
			if listItem.Item.IsEmpty() {
				continue
			}
			if readable {
				fsizeValue = BytesToReadable(listItem.Item.Fsize)
			} else {
				fsizeValue = listItem.Item.Fsize
			}
			if notfilterSuffix && notfilterTime {
				lineData := fmt.Sprintf("%s\t%v\t%s\t%d\t%s\t%d\t%s\r\n",
					listItem.Item.Key, fsizeValue, listItem.Item.Hash,
					listItem.Item.PutTime, listItem.Item.MimeType, listItem.Item.Type, listItem.Item.EndUser)
				_, wErr := bWriter.WriteString(lineData)
				if wErr != nil {
					retErr = wErr
					errorWarning(lastMarker, retErr)
				}

			} else {
				var hasSuffix = true
				var putTimeValid = true

				if !notfilterTime { // filter by putTime
					putTime := time.Unix(listItem.Item.PutTime/1e7, 0)
					putTimeValid = filterByPuttime(putTime, startDate, endDate)
				}
				if !notfilterSuffix {
					key := listItem.Item.Key
					hasSuffix = filterBySuffixes(key, suffixes)
				}

				if hasSuffix && putTimeValid {
					lineData := fmt.Sprintf("%s\t%v\t%s\t%d\t%s\t%d\t%s\r\n",
						listItem.Item.Key, fsizeValue, listItem.Item.Hash,
						listItem.Item.PutTime, listItem.Item.MimeType, listItem.Item.Type, listItem.Item.EndUser)
					_, wErr := bWriter.WriteString(lineData)
					if wErr != nil {
						retErr = wErr
						errorWarning(lastMarker, retErr)
					}
				}
			}
		}
		fErr := bWriter.Flush()
		if fErr != nil {
			retErr = fErr
			errorWarning(lastMarker, retErr)
			if maxRetry > 0 {
				c++
			}
		}
		if lastMarker == "" {
			break
		} else {
			marker = lastMarker
		}
	}

	return
}
