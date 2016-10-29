sync
---------------

Go中`sync`包下有2种`mutex`实现：

* `sync.Mutex`

* `sync.RWMutex`

`Mutex`底层基于`sync/atomic`实现了
[Compare and Swap](https://en.wikipedia.org/wiki/Compare-and-swap).
由于该算法逻辑只需要一条汇编就可以实现，在单核CPU上运行是可以保证原子性的，但多
核CPU上运行时，需要加上`LOCK`前缀来对总线加锁，从而保证了该指令的原子性：

```
// src/sync/atomic/asm_amd64.s#L35
TEXT ·CompareAndSwapInt32(SB),NOSPLIT,$0-17
	JMP	·CompareAndSwapUint32(SB)

TEXT ·CompareAndSwapUint32(SB),NOSPLIT,$0-17
  // 初始化参数
	MOVQ	addr+0(FP), BP
	MOVL	old+8(FP), AX
	MOVL	new+12(FP), CX
  // 锁总线
	LOCK
  // 执行Compare and Exchange
	CMPXCHGL	CX, 0(BP)
  // 处理返回值
	SETEQ	swapped+16(FP)
	RET
```

`sync.Mutex`实现了`sync.Locker`接口，主要有`Lock()/Unlock()`2个method，值得注意
的是，不能重复的对已经解锁的mutex解锁，否则会`panic`.

`sync.Mutex`阻塞进程的方式其实是让进程不断的轮询.

值得注意的是`Mutex`的`zero value`是一个`unlocked mutex`:

```
// A Mutex is a mutual exclusion lock.
// Mutexes can be created as part of other structures;
// the zero value for a Mutex is an unlocked mutex.
//
// A Mutex must not be copied after first use.
```

在加锁和解锁的时候，都会检查是否`enabled race`，允许抢占可能会造成不可预知的问题
看起来只有在`sync.RWMutex`和`sync.WaitGroup`里面才使用了`race.Enable()`，目前
还不清楚这个东西的作用是什么.

Go1.6中引入了更可靠的竞争检测机制，
[Introducing the Go Race Detector](https://blog.golang.org/race-detector).
只要执行`Go Command`的时候带上`-race`即可：

例4.1 `concurrency/e4_1.go` 在没有加锁的情况下存在多个`goroutine`同时读写公有
数据

```
go run -race concurrency/e4_1.go

...
Found 2 data race(s)
```

另外, `sync.RWMutex`，同样实现了`sync.Locker`接口，提供了更灵活的锁机制：

```
// An RWMutex is a reader/writer mutual exclusion lock.
// The lock can be held by an arbitrary number of readers or a single writer.
// RWMutexes can be created as part of other structures;
// the zero value for a RWMutex is an unlocked mutex.
//
// An RWMutex must not be copied after first use.
//
// If a goroutine holds a RWMutex for reading, it must not expect this or any
// other goroutine to be able to also take the read lock until the first read
// lock is released. In particular, this prohibits recursive read locking.
// This is to ensure that the lock eventually becomes available;
// a blocked Lock call excludes new readers from acquiring the lock.
```

有意思的是，`sync.RWMutex`会检测公有数据的修改；
例4.2 `concurrency/e4_2.go`中模拟了2组`goroutine`，一组专门读数据，一组专门
写数据；其中读数据的速度非常快，写数据会比较慢（加了随机延迟）：

```
package main

import (
	"math/rand"
	"sync"
	"time"
)

var shared = struct {
	*sync.RWMutex
	count int
}{}

var wg *sync.WaitGroup

const N = 10

func main() {
	rand.Seed(time.Now().Unix())
	shared.RWMutex = new(sync.RWMutex)
	wg = new(sync.WaitGroup)
	wg.Add(2 * N)
	defer wg.Wait()

	for i := 0; i < N; i++ {
		// write goroutines
		go func(ii int) {
			shared.Lock()
			duration := rand.Intn(5)
			// shared.Lock()
			time.Sleep(time.Duration(duration) * time.Second)
			shared.count++
			println(ii, "write --- shared.count =>", shared.count)
			// shared.Unlock()
			shared.Unlock()

			wg.Done()
		}(i)

		// read goroutines
		go func(ii int) {
			shared.RLock()
			println(ii, "read --- shared.count =>", shared.count)
			shared.RUnlock()

			wg.Done()
		}(i)

	}
}
```

读写锁的效果就是，在写锁锁定的时候，会阻塞所有的读锁，
例4.2里面读操作非常快，所以第一次写操作完成之后，
所有的读操作就一次完成了，后面的就只有读操作了

例4.3 `concurrency/e4_3.go` 去掉了公有数据的读写操作，
模拟了例4.2里面的延迟和锁；本来以为锁会检查逻辑里面的数据修改，发现并不是；
看起来写锁对读锁的阻塞是全局的，只要一个进程内的写锁就会阻塞所有的读锁；

从底层实现上来看，上面的判断应该是准确的:

```
// src/sync/rwmutex.go#L35
// RLock locks rw for reading.
func (rw *RWMutex) RLock() {
	if race.Enabled {
		_ = rw.w.state
		race.Disable()
	}
	if atomic.AddInt32(&rw.readerCount, 1) < 0 {
		// A writer is pending, wait for it.
		runtime_Semacquire(&rw.readerSem)
	}
	if race.Enabled {
		race.Enable()
		race.Acquire(unsafe.Pointer(&rw.readerSem))
	}
}
```

操作系统只对`goroutine`实际使用的系统线程分配资源，而且`goroutine`实现了进程中
上线文切换机制，比操作系统级切换系统线程上下文高效；尽管这样，并发过高的情况下
`goroutine`之间上下文切换也会对程序性能带来比较大的影响;

