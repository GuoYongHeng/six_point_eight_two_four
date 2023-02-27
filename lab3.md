## 相关资料


### lab3介绍
在这个实验中你将使用lab2中的raft库实现一个容错的key/value存储服务。你的key/value存储服务将是一个复制状态机，由多个使用raft进行复制的key/value服务组成。只要大多数服务器处于活动状态并且可以通信，你的key/value服务就应该继续处理客户端请求，尽管存在其他故障或网络分区。在完成lab3之后，你将实现[raft交互图](https://pdos.csail.mit.edu/6.824/notes/raft_diagram.pdf)的所有部分（clerk，service和raft）。

客户端可以发送三个不同的rpc到key/value服务：Put(key, value),Append(key, arg)和Get(key)。服务维护了一个简单的key/value数据库。键和值是字符串。Put(key, value)替换数据库中特定键的值。Append(key, arg)将arg添加到键的值，Get(key)获取键的当前值。一个不存在的键Get应该返回空字符串。Append一个不存在的键应该像Put一样。每个客户端通过Clerk使用Put/Append/Get方法与服务沟通。Clerk管理与服务器的RPC交互。

你的服务必须安排应用通过Clerk调用Get/Put/Append函数的顺序是可序列化的。如果一次调用一个Get/Put/Append函数应该表现得好像系统只有他的状态的一个副本，并且每个调用应该都应该观察到对前面调用序列所隐含的状态的修改。对于并发调用，返回值和最终状态必须与操作按照某种顺序执行一次得到的结果相同。如果调用在时间上重叠则调用是并发的：例如，如果客户端X调用Clerk.Put()，并且客户端Y调用Clerk.Append()，然后客户端X的调用返回。一个调用必须观察到在它调用开始之前所有已完成的调用的效果。

线性化对于应用程序来说是方便的，因为它是你从一次处理一个请求的单个服务器上看到的行为。例如，如果一个客户端从服务端获得更新请求的成功响应，则保证随后其他客户端启动的读取可以看到该更新的效果。对于单服务器来说提供线性化是相当容易的。如果服务器是复制的，那就更难了，因为所有的服务器必须为并发请求选择同样的执行顺序。必须避开使用不是最新的状态去回复客户端，并且必须在失败后以一种保留所有确认的客户端更新的方式去恢复它们的状态。

这个实验有两个部分。在A部分，你将会使用你实现的Raft实现一个key/value服务，但是不会使用到snapshots。在B部分，你将会使用你在实验2D实现的的snapshot，这允许你的Raft抛弃旧的日志条目。请在各自的截止日期之前提交每个部分。

你应该继续看[论文](https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf)，特别是第七章和第八章。想要获得更宽广的视角，可以查看Chubby，Paxos Made Live，Spanner，Zookeeper，Harp，Viewstamped Replication和Blolosky等。

### 开始
我们在src/kvraft中为你提供框架代码，你需要去修改kvraft/client.go，kvraft/server.go，也许还有kvraft/common.go。

要启动并运行，请执行以下命令。不要忘记git pull去获取最新的代码。
```shell
$ cd ~/6.5840
$ git pull
...
$ cd src/kvraft
$ go test
...
$
```

#### A部分：没有Snapshots的Key/Value服务
你的每个key/value服务器("kvservers")都有一个对应的Raft对等体。Clerk发送Put(),Append(),和Get() RPC到其关联Raft是领导的kvserver。kvserver代码将会提交Put/Append/Get到Raft，这样Raft日志就会保存一系列Put/Append/Get操作。所有的kvserver按顺序从Raft日志执行操作，将操作应用到他们的键值数据库，目的是让服务器维护键值数据库的相同副本。

Clerk有时候不知道哪个kvserver是Raft leader。如果Clerk发送一个RPC到错误的kvserver，或者无法到达kvserver，Clerk应该通过发送到不同的kvserver来重试。如果键值服务将操作提交到Raft日志（并且因此将操作应用于键值状态集），领导者通过相应其RPC将结果报告给Clerk。如果操作提交失败（例如，如果leader被替换），服务器会报告错误，并且Clerk会尝试使用不同的服务器。

你的kvserver不应该直接通信，他们应该只通过Raft交互。

#### 任务
你的首要任务是实现一个解决方案，该方案在没有丢失消息且没有发生故障时可以正常工作。

你需要添加rpc发送代码到client.go中的Clerk Put/Append/Get，并且在server.go中实现PutAppend()和Get() RPC处理程序。这些处理程序应该使用Start()在Raft log中输入一个Op，你应该在server.go中填充结构定义以便用来描述Put/Append/Get操作。每一个服务器都应该执行Op命令，即当他们出现在applyCh上的时候。一个RPC处理程序应该注意到Raft何时提交其Op，然后回复RPC。

当你可靠的通过测试中的第一个测试："One client"时，你就完成了这个任务
#### 提示
* 在调用Start()之后，你的kvserver将需要去等待Raft去完成协议。已经达成一致的命令到达applyCh。当PutAppend()和Get()处理程序使用Start()提交命令到Raft log时，你的代码将会保持读取applyCh。当心kvserveer和Raft库之间的死锁。
* 你被允许向Raft的ApplyMsg中添加字段，也被允许想Raft RPC中添加字段，例如AppendEntries，但是这对于大多数实现来说是不必要的。
* 如果kvserver不是大多数的一部分，则它不应该完成Get() RPC（这样他就不会提供陈旧数据）。一个简单的解决方案是在Raft log中进入每个Get()（以及每个Put()和Append()）。你不必实现第8节描述的只读优化操作。
* 最好在一开始就加锁，因为避免死锁的需求又是会影响整个代码设计。使用go test -race检查你的代码是否没有竞争。

现在你应该修改你的解决方案以在网络和服务器故障时继续。你将面临的一个问题是Clerk可能必须要发送一个RPC多次，知道它找到肯定答复的kvserver。如果一个leader在向Raft log提交一个条目后故障了，则Clerk可能不会受到恢复，并且因此可能会重新发送请求到其他的leader。每次调用Clerk.Put()或者Clerk.Append()应该只执行一次,因此你必须确保重新发送请求不会导致服务器执行请求两次。

#### 任务
添加代码去处理故障，并且处理重复的Clerk请求，包括Clerk在一个任期向kvserver领导者发送请求，等待回复超时，并且在另一个任期重新向新的kvserver leader发送请求的情况。请求应该只被执行一次。你的代码应该通过go test -run 3A测试。

#### 提示
* 你的解决方案需要处理已被Clerk的RPC调用Start()的领导者，但是在请求提交到log之前失去了领导地位。在这种情况下，你应该安排Clerk重新发送请求到其他的服务器，直到发现新的leader。这样做的一种方法是，服务器通过注意到Start()返回的索引中出现了不同的请求，或者Raft的term发生了变化来检测到他失去了领导地位。如果前任领导人自己被分区，他将不会知道新的leader，但是统一分区中的任意客户端有我都无法与新的leader通信，因此在这种情况下，服务器和客户端可以无限期的等待知道分区恢复正常。
* 你可能必须修改你的Clerk去记住哪个服务器是最后一个RPC的领导者，并首先将下一个RPC发送到该服务器。这将避免浪费时间在每个RPC上搜索leader，这可能会帮助你足够快速的通过一些测试。
* 你将需要唯一表示客户端操作以确保key/value服务对每个操作只执行一次。
* 你的重复检测方案应该快速释放服务器内存，例如通过每个RPC暗示客户端已经看到了之前RPC的恢复。可以假设客户同一时间只会调用一次Clerk。

你的代码现在应该通过了实验3A的测试，就像下边这样：
```shell
$ go test -run 3A
Test: one client (3A) ...
  ... Passed --  15.5  5  4576  903
Test: ops complete fast enough (3A) ...
  ... Passed --  15.7  3  3022    0
Test: many clients (3A) ...
  ... Passed --  15.9  5  5884 1160
Test: unreliable net, many clients (3A) ...
  ... Passed --  19.2  5  3083  441
Test: concurrent append to same key, unreliable (3A) ...
  ... Passed --   2.5  3   218   52
Test: progress in majority (3A) ...
  ... Passed --   1.7  5   103    2
Test: no progress in minority (3A) ...
  ... Passed --   1.0  5   102    3
Test: completion after heal (3A) ...
  ... Passed --   1.2  5    70    3
Test: partitions, one client (3A) ...
  ... Passed --  23.8  5  4501  765
Test: partitions, many clients (3A) ...
  ... Passed --  23.5  5  5692  974
Test: restarts, one client (3A) ...
  ... Passed --  22.2  5  4721  908
Test: restarts, many clients (3A) ...
  ... Passed --  22.5  5  5490 1033
Test: unreliable net, restarts, many clients (3A) ...
  ... Passed --  26.5  5  3532  474
Test: restarts, partitions, many clients (3A) ...
  ... Passed --  29.7  5  6122 1060
Test: unreliable net, restarts, partitions, many clients (3A) ...
  ... Passed --  32.9  5  2967  317
Test: unreliable net, restarts, partitions, random keys, many clients (3A) ...
  ... Passed --  35.0  7  8249  746
PASS
ok  	6.5840/kvraft	290.184s
```
每个Passed后面的数字是实时秒数，对等点数，发送的RPC数（包括客户端RPC），和执行的key/value操作数（Clerk Get/Put/Append调用）。


#### B部分：带快照的Key/Value服务
就目前情况而言，你的key/value服务器还不能调用你的Raft库的Snapshot()函数，因此重启服务器将会重放完整持久化的raft log来使其恢复状态。现在你将会修改kvserver以与Raft协作节省log控件，并且使用lab 2D中的Raft的Snapshot()来减少重启时间。

测试者将maxraftstate参数传递到你的StartKVServer()中。maxraftstate指明你的持久化Raft状态的字节数限制（包括日志，但是不包括快照）。你应该对比maxraftstate和persister.RaftStateSize()。无论何时当你的key/value服务检测到Raft状态的尺寸接近阈值时，它应该调用Raft的Snapshot来保存快照。如果maxraftstate为-1，则不需要进行快照。maxraftstate适用于作为第一个参数传递给persister.Save()的GOB编码字节。

#### 任务
修改你的kvserver，以便它可以检测到Raft状态何时增长的太大，然后让Raft创建快照。当一个kvserver重启时，它应该从persister读取快照并从快照恢复状态。

#### 提示
* 思考何时kvserver应该创建对其状态进行快照并且快照应该包含什么。Raft使用Save()将每个快照连同相应的Raft状态存储的可持久化对象中。你可以使用ReadSnapshot()读取最新存储的快照。
* 你的kvserver必须能够跨检测点探测到日志中的重复操作，因此你用于检查它们的任何状态都必须包含在快照中。
* 将存储快照的结构的所有字段大写
* 你的Raft库可能在本实验中暴露错误。如果你对Raft实现进行更改，请确保他们仍然可以通过Lab 2中的测试。
* Lab 3测试的合理时间是400秒的显示时间和700秒的CPU时间。此外，go test -run TestSnapshotSize花费的现实时间应该小于20秒。

你的代码应该通过3B测试（如此处例子所示）以及3A测试（并且你的Raft必须仍然可以通过Lab 2的测试）。
```shell
$ go test -run 3B
Test: InstallSnapshot RPC (3B) ...
  ... Passed --   4.0  3   289   63
Test: snapshot size is reasonable (3B) ...
  ... Passed --   2.6  3  2418  800
Test: ops complete fast enough (3B) ...
  ... Passed --   3.2  3  3025    0
Test: restarts, snapshots, one client (3B) ...
  ... Passed --  21.9  5 29266 5820
Test: restarts, snapshots, many clients (3B) ...
  ... Passed --  21.5  5 33115 6420
Test: unreliable net, snapshots, many clients (3B) ...
  ... Passed --  17.4  5  3233  482
Test: unreliable net, restarts, snapshots, many clients (3B) ...
  ... Passed --  22.7  5  3337  471
Test: unreliable net, restarts, partitions, snapshots, many clients (3B) ...
  ... Passed --  30.4  5  2725  274
Test: unreliable net, restarts, partitions, snapshots, random keys, many clients (3B) ...
  ... Passed --  37.7  7  8378  681
PASS
ok  	6.5840/kvraft	161.538s
```