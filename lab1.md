
## 相关资料
[MapReduce论文阅读](https://tanxinyu.work/mapreduce-thesis/)

## lab1工作内容

你的任务是实现一个分布式的MapReduce，MapReduce由两个程序组成，分别是coordinator和worker。这里实现的MapReduce只有一个coordinator进程，有一个worker进程或者多个并行的worker进程。在现实世界中worker将会运行在一堆机器上，但是在这个lab中你将在一台机器上运行worker进程。worker进程通过rpc和coordinator进程通信。**每一个worker进程都会向coordinator进程请求任务**，从一个或多个文件中读取任务的输入，执行任务然后将输出写到一个或多个文件中。coordinator进程应该可以注意到没有在规定时间（该lab为10s）内完成任务的worker进程，然后将该任务给分配给另一个不同的worker进程。

我们已经给了你一部分启动代码。coordinator和worker的主要程序分别在main/mrcoordinator.go和main/mrworker.go中，不要修改这两个文件。你应该将你的代码放在mr/coordinator.go，mr/worker.go和mr/rpc.go中。

这里是怎么样去运行word-count MapReduce应用。首先，确保word-count插件是刚刚构建的：
```shell
$ go build -buildmode=plugin ../mrapps/wc.go
```
在main目录中，运行coordinator
```shell
$ rm mr-out*
$ go run mrcoordinator.go pg-*.txt
```
pg-*参数是mrcoordinator.go中的输入文件，每一个文件作为一个Map任务的输入。

在其他的终端中，运行worker进程：
```shell
$ go run mrworker.go wc.so
```
当worker和coordinator结束时，从文件mr-out-*查看程序输出。当你完成lab的时候，输出文件的排序的并集应该和顺序输出是匹配的，就像这样：
```shell
$ cat mr-out-* | sort | more
A 509
ABOUT 2
ACT 8
...
```

我们为你提供了测试脚本main/test-mr.sh。这些测试检查wc和indexer MapReduce应用在给定pg-xxx.txt文件作为输入时是否产生正确的输出。这些测试也检查你的程序是否并行运行Map和Reduce任我游，并且还检查你的程序是否可以恢复运行时崩溃的worker。

如果你现在运行测试脚本，他将会挂起，因为coordinator永远不会结束。
```shell
$ cd ~/6.824/src/main
$ sh test-mr.sh
*** Starting wc test.
```

你可以在mr/coordinator.go中修改`ret:=false`为true来结束coordinator。然后：
```shell
$ sh ./test-mr.sh
*** Starting wc test.
sort: No such file or directory
cmp: EOF on mr-wc-all
--- wc output is not the same as mr-correct-wc.txt
--- wc test: FAIL
$
```

对于每个reduce任务，这个测试脚本期望在名为mr-out-X文件中看到输出。没有实现相应功能的mr/coordinator.go和mr/worker.go不会生成这些文件，所以此时测试会失败。

当程序运行结束后，测试脚本的输出如下：
```shell
$ sh ./test-mr.sh
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting crash test.
--- crash test: PASS
*** PASSED ALL TESTS
$
```

你将会看到一些来自Go RPC包错误，如下：
2019/12/16 13:27:09 rpc.Register: method "Done" has 1 input parameters; needs exactly three
忽略这些信息。

### 一些规则
* map阶段应该将中间键划分到nReduce个reduce任务的桶中，其中nReduce是main/mrcoordinator.go传到MakeCoordinator()中的参数。因此每个map需要创建nReduce个中间文件供reduce任务使用。
* worker的实现应该将第X个reduce任务的输出放在文件mr-out-X中
* 一个mr-out-X文件应该包含一行每个Reduce函数的输出。该行使用Go中的"%v %v"格式，通过key和value来调用。可以查看main/mrsequential.go中注释为"this is the correct format"的行。如果你实现的代码中的输出格式与该格式偏离太多，测试脚本将会失败
* 你可以修改mr/worker.go，mr/coordinator.go和mr/rpc.go。您可以临时修改其他文件进行测试，但请确保您的代码适用于原始版本；我们将使用原始版本进行测试。
* worker应该将中间的Map输出放在当前目录的文件中，稍后worker可以读取他们作为Reduce任务的输入。
* main/mrcoordinator.go期待mr/coordinator.go去实现一个Done()函数可以在MapReduce job完成的时候返回true，同时，mrcoordinator.go将会退出。
* 当job完成后，worker进程应该退出。实现该功能的一个简单方法是使用call()的返回值：如果worker无法联系到，它可coordinator以假设coordinator已经退出了，因为job已经完成，因此worker也可以终止。根据你的设计，你可能会发现去coordinator发送给worker一个"please exit"的伪任务是有用的。


### 提示
* [指导页](https://pdos.csail.mit.edu/6.824/labs/guidance.html)有一些开发和调试技巧
* 开始的一种方法是修改mr/worker.go中的Worker()函数通过RPC向coordinator发送请求获取任务。然后修改coordinator，令其回复还没经过map函数处理的文件的文件名。然后修改worker去读取文件然后调用Map函数，就像mrsequential.go中那样。
* 应用程序 Map 和 Reduce 函数在运行时使用 Go 插件包从名称以 .so 结尾的文件中加载。
* 如果你更改了mr目录下的任何内容，你可能要必须重新构建你使用的MapReduce插件，就像`go build -buildmode=plugin ../mrapps/wc.go`
* 该lab依赖于worker都共享一个文件系统。当所有的worker都工作在一台机器上，这是很简单的，但是如果worker运行在不行的机器上，将会需要一个像GFS这样的全局文件系统
* 中间文件的合理命名约定是mr-X-Y，X是Map任务的编号，Y是reduce任务的编号。
* worker的Map任务将需要一个方法去将中间key/value对存储到文件中，这个方法可以在reduce任务期间正确的读取文件。一个可能的方法是使用Go的encoding/json包。将key/value对写入到json文件：

    ```go
    enc := json.NewEncoder(file)
    for _, kv := ... {
        err := enc.Encode(&kv)
    ```
    读取文件如下：
    ```go
    dec := json.NewDecoder(file)
    for {
        var kv KeyValue
        if err := dec.Decode(&kv); err != nil {
        break
        }
        kva = append(kva, kv)
    }
    ```
* 你的worker中的map部分可以使用ihash(key)函数来根据给定的key选择reduce任务
* 你可以从mrsequential.go文件中查看编码过程中用到的一些功能的代码
* coordinator作为rpc服务器是并行的，不要忘记对共享数据加锁
* `go build -race`和`go run -race`使用Go的rece detector。test-mr.sh中默认使用race detector。
* worker进程有时需要等待，在最后一个map任务结束之前，reduce任务不能启动。一种可能的方式是worker周期性的向coordinator发起请求请求任务，在每次请求之间可以使用`time.Sleep()`睡眠。另一个可能的方式是coordinator相关的rpc处理程序中有一个等待的循环，该循环等待`time.Sleep()`或者`sync.Cond`。Go在其自己的线程中为每个rpc运行处理程序，因此一个处理程序等待的时候并不会阻塞coordinator进程处理其他rpc。
* coordinator无法可靠的区分崩溃的worker、还活着但是由于一些原因停止的worker和正在执行但是执行速度太慢而无用的worker。你能做的最好的事情就是让coordinator等待一些时间，然后放弃并将任务重新发给不同的worker。对于这个lab，coordinator等待10s，之后coordinator应该假设worker已经死掉了（当然这时worker可能并没有死掉）
* 如果你选择去实现备份任务（3.6节），请注意worker执行任务且不崩溃时，你的代码不能调度额外的任务。备份任务应该只安排在一些相对较长的时间周期之后（例如10s）。
* 为了测试从崩溃中恢复，你可以使用mrapps/crash.go插件，它随机的退出Map和Reduce函数
* 为了确保在崩溃时没有人可以观测到部分写入的文件，MapReduce论文提到了使用临时文件并在完全写入后自动重命名它的技巧。你可以使用`ioutil.TempFile`去创建一个临时文件并且用`os.Rename`去自动给它重命名。
* test-mr.sh在子目录mr-tmp运行所有的进程，因此如果出现问题并且你想查看中间文件或输出文件，可以查看那里。你也可以修改test-mr.sh使其在测试失败时就退出，这样就奔就不会继续测试（并覆盖输出文件）。
* test-mr-many.sh提供了一的简单的脚本，用于运行带有超时的test-mr.sh。他以运行测试的次数作为参数。你不应该并行运行多个test-mr.sh实例，因为coordinator会重复使用相同的socket，从而导致冲突。
* Go RPC 仅发送名称以大写字母开头的结构字段。子结构也必须有大写的字段名称。
* 当传送reply结构的指针到rpc的时候，*reply对象应该是未分配的。rpc调用的代码应该总是像下边这样。
    ```
      reply := SomeType{}
      call(..., &reply)
    ```
    在调用之前不设置reply的任何字段。如果你不遵循这个要求，当你预初始化一个reply字段为非默认值并且执行rpc的服务器设置reply字段为默认值的时候将会出现问题。你将会观察到写入似乎并没有效果，并且在调用方，非默认值仍然存在。