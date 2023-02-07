## 相关资料
[GFS论文阅读](https://tanxinyu.work/gfs-thesis/)

[VMware FT论文阅读](https://tanxinyu.work/vm-ft-thesis/)

[Raft算法介绍](https://tanxinyu.work/raft/)

[Raft论文相关](https://tanxinyu.work/raft-thesis-translate/)

### lab2介绍
这是一系列实验中的第一个，在该实验中你将构建一个容错的key/value存储系统。在这个实验中，你将实现raft，一种复制状态机协议。在下个实验中，你将在raft之上构建一个key/value服务。然后你将在多个复制状态机上"shard"你的服务以获得更高的性能。

复制服务通过存储在多个副本服务器上存储其状态（数据）的完整副本来实现容错。复制允许服务继续运行，即使一些服务出现了故障（崩溃或者损坏或者不稳定的网络）。挑战在于故障可能会导致副本持有不同的数据拷贝。

Raft将客户端请求组织成一个序列，叫做log，并且确保所有的副本服务器看到相同的log。每个副本服务器按照日志顺序执行客户端请求，并将他们应用到服务状态的本地副本。由于所有的活着的副本会看到同样的log内容，他们都以相同的顺序执行相同的请求，因此他们都继续具有相同的服务状态。如果一个服务器出现故障但是后来恢复了，Raft会负责更新他的日志。只要大多数服务器还活着并且能够互相通信，Raft就会继续运行。如果没有那么多活着的服务器，Raft将不会继续运行，但一旦大多数服务器可以再次通信，Raft就会从中断的地方继续运行。

在本实验中，你会将Raft实现为一个具有关联方法的Go对象，旨在用作更大的服务中的模块。一组Raft实例通过RPC通信去维护日志副本。你的Raft接口将支持一个不确定的编号命令序列，也叫做日志条目。这些条目用index numbers编号。具有给定索引的日志提哦啊木最终将被提交。那时，你的Raft应该发送日志条目到更大的服务以供其执行。

你应该遵循extended raft paper中的设计，特别注意图2。你将会实现论文中的大部分内容，包括保存持久状态和在节点出现故障并重启后读取它。你将不会实现集群成员更改（section 6）。你将会在之后的实验中实现日志压缩/快照（section 7）

你可能会发现这个指南以及这个关于并发的locking和structure的建议是有用的。从更广泛的角度来看，你还要看一下Paxos，Chubby，Paxos Made Live，Spanner，Zookeeper，Harp，Viewstamped Replication和Bolosky等。

### 开始
我们为你提供框架代码src/raft/raft.go。我们还提供了一组测试，你应该使用它们来推动你的实施工作，我们将使用他们来对你提交的实验进行评分。这些测试在src/raft/test_test.go中。

要启动并运行，请执行下边的命令。不要忘记`git pull`去获取最新的软件。
```shell
$ cd ~/6.824
$ git pull
...
$ cd src/raft
$ go test
Test (2A): initial election ...
--- FAIL: TestInitialElection2A (5.04s)
        config.go:326: expected one leader, got none
Test (2A): election after network failure ...
--- FAIL: TestReElection2A (5.03s)
        config.go:326: expected one leader, got none
...
$
```

### 代码
通过添加代码到raft/raft.go来实现Raft。在那个文件中你将会发现框架代码，以及如何发送和接受RPC的示例。

你的实现必须支持以下接口，测试器和你的key/value服务器将会使用。你将会在raft.go的注释中发现更多细节。
```go
// create a new Raft server instance:
rf := Make(peers, me, persister, applyCh)

// start agreement on a new log entry:
rf.Start(command interface{}) (index, term, isleader)

// ask a Raft for its current term, and whether it thinks it is leader
rf.GetState() (term, isLeader)

// each time a new entry is committed to the log, each Raft peer
// should send an ApplyMsg to the service (or tester).
type ApplyMsg
```
一个服务调用`Make(peers,me,...)`去创建一个Rafr peer。peers参数是Raft对等点（包括当前这个）的网络标识符数组，用于RPC。me参数是当前这个节点在peers数组中的索引。`Start(command)`要求Raft开始添加命令到副本日志的结尾的处理。`Start`应该立即返回，不需要等待日志追加完成。服务希望你的视线为每个新提交的日志条目发送一个`ApplyMsg`到`Make()`的`applyCh`参数。

raft.go包含发送RPC(`sendRequestVote()`)和处理传入RPC(`RequestVote()`)的示例代码。你的Raft peers应该使用labrpc Go包来交换RPC(源码在src/labrpc)。测试者可以告诉labrpc去延迟RPC信息，重新排序，并丢弃它们来模拟各种各样的网络故障。同时你可以临时修改labrpc，但确保你的Raft工作在原始的labrpc上，因为我们将使用它来测试和评分你的lab。你的Raft实例必须只和RPC交互，例如，他们不允许使用共享变量或文件来进行交流。

后续实验建立在这个实验的基础上，所以给你自己足够的时间去编写可靠的代码是很重要的。

### Part 2A:leader election

#### 任务
实现Raft leader选举和心跳(不带log entries的AppendEntries RPC)。Part 2A的目标是选举出一个单一的leader；如果没有故障，则leader仍然是leader；如果旧的leader出现故障或者旧的leader收发的包丢失，新的leader要去接管旧的。运行`go test -run 2A`去测试你的2A的代码

#### 提示
* 你不能简单的直接去运行你的Raft代码，你应该通过tester来运行他，即`go test -run 2A`
* 按照论文中的图2，在这里你应该惯出发送和接收RequestVote、与玄驹相关的服务器规则和与leader选举相关的状态
* 在raft.go中的Raft结构中为leader选举添加图2的状态。你也需要定义一个结构去保存和每个log entry相关的信息
* 填充`RequestVoteArgs`和`RequestVoteReply`结构。修改`Make()`去创建一个后台协程，该协程在一段时间内没有收到其他对等节点的消息时将会通过发送`RequestVote`来定期开始leader选举。通过这种方式peer将会知道谁是leader，如果已经存在leader的话，或者它自己会编程leader。实现`RequestVote()`RPC函数以便服务器互相投票
* 去实现心跳机制。定义一个`AppendEntries`RPC结构（尽管你可能还不需要所有的参数），并且leader会定期发送他们。编写一个`AppendEntries`RPC处理函数来重设选举超时，以便其他的服务器不会在已经有leader的情况下再成为leader。
* 确保不同peer的选举超时不会总是同时触发，否则所有peer都只会为自己投票并且没有节点会成为leader
* tester要求leader每秒发送心跳不超过10次
* tester要求你的Raft在旧的leader故障后5秒内选出新的leader（如果大多数节点仍然可以通信）。但是请记住，leader选举可能需要多轮以防止分裂投票（如果数据包丢失或者候选leader不行选择了相同的随机退避时间，就会发生这种情况）。你必须选择一个足够短的选举超时（以及心跳间隔），这样选举才会在5秒内完成，即使需要多轮选举。
* 论文的5.2节提到了150到300ms的选举超时。只有当leader发送心跳包的频率超过150ms每次时，这个范围才有意义。因为tester限制你每秒10次心跳，所以你必须使用大于论文中150到300ms的选举超时，但是不要太大，因为那样的话你可能无法在5秒内选出leader。
* 你可能会发现Go的rand很有用
* 你需要编写周期的或者在延迟一定时间后的采取动作的代码。最简单的方法是创建一个协程在里面循环调用time.Sleep()（请参阅Make()为了这个目的创建的ticker()协程）。不要使用Go的time.Timer或者time.Ticker，他们很难正确使用。
* [指导页面](https://pdos.csail.mit.edu/6.824/labs/guidance.html)有一些关于如何开发和调试代码的提示。
* 如果你的代码无法通过测试，请再次阅读论文的图2，leader选举的逻辑分布在图中的多个部分。
* 不要忘记实现`GetState()`
* tester在永久关闭一个实例的时候会调用你的Raft的rf.Kill()。你可以使用rf.Kill()检测Kill()是否被调用。你可能希望在所有的循环中执行这个操作，以避免死掉的Raft实例打印出令人困惑的消息。
* Go RPC仅发送以大写字母开头的结构字段。子结构也必须具有大写的字段名称（例如一个数组中的日志记录）。labgob包将会将告你这一点，不要忽略警告。

确保你在提交Part 2A之前通过了2A测试，这样你就会看到如下内容：
```shell
$ go test -run 2A
Test (2A): initial election ...
  ... Passed --   3.5  3   58   16840    0
Test (2A): election after network failure ...
  ... Passed --   5.4  3  118   25269    0
Test (2A): multiple elections ...
  ... Passed --   7.3  7  624  138014    0
PASS
ok  	6.5840/raft	16.265s
$
```
每一个"Passed"行包含五个数字，这些是测试花费的时间（以秒为单位）、Raft peers的数量、在测试期间发送RPCS的次数、RPC信息的总字节数和Raft上报的已提交的日志条目的数量。你测试时显示的数字将和这里的不一样。如果愿意，你可以忽略这些数字，但是他们可以帮助你全面检查你的实现中发送的RPC的数量。对于所有的lab2，3，4，如果所有测试花费的时间超过了600秒，或者任何单个测试花费时间超过120s，成绩脚本将会拒绝你的解决方案。

当我们对你提交的内容进行评级的时候，我们运行测试的时候将不会带上`-race`，但是你应该确保你的代码可以通过带有`-race`的测试。

### Part 2B:log

#### 任务
实现leader和follower的添加新的日志条目的代码，以便`go test -run 2B`测试的通过

#### 提示
* 运行`git pull`以获得最新的lab software
* 你的第一个目标应该是通过TestBasicAgree2B()。通过实现Start()开始，然后编写代码通过`AppendEntries` RPCs来发送和接收新的日志条目，如图2所示。在每个对等端的`applyCh`上发送每个新的已提交的条目。
* 你需要实现选举限制(论文中的5.4.1节)
* 在早期的Lab 2B测试中无法达成一致的一种方法是即使领导者还活着，也要进行重复选举。寻找选举定时器管理中的bug，或者在赢得选举后不要立即发送心跳。
* 你的代码可能包含重复检查某些事件的循环。不要让这些循环连续不间断的执行，因为那将会减慢你的实现的速度，以至于无法通过测试。使用Go的<u>condition variables</u>，或者在每次循环迭代中插入`time.Sleep(10 * time.Millisecond`
* 为你之后的lab帮个忙，编写干净清晰的代码。如果需要一些想法，可以重新访问[Guidance page](https://pdos.csail.mit.edu/6.824/labs/guidance.html)，其中包含怎么开发和调试代码。
* 如果你没有通过测试，查看config.go和test_test.go的代码，可以更好地理解测试正在测试什么。config.go中也说明了tester怎么使用Raft API

### Part 2C:persistence

#### 任务

#### 提示

### Part 2D:log compaction

#### 任务

#### 提示

## State
|Persistent state on all servers|Description|
|---|---|
|currentTerm|---|
|votedFor|---|
|log[]|---|

|Volatile state on all servers|Description|
|---|---|
|commitIndex|---|
|lastApplied|---|

|Volatile state on leaders|Description|
|---|---|
|nextIndex[]|nextIndex的初始值从新任Leader的最后一条日志开始|
|matchIndex[]|---|

## AppendEntries RPC
|Arguments|Description|
|---|---|
|term|---|
|leaderId|---|
|prevLogIndex|nextIndex[i]-1所指位置的日志的索引|
|prevLogTerm|上边那个日志所属的Term|
|entries[]|---|
|leaderCommit|---|

|Results|Description|
|---|---|
|term|---|
|success|---|

|Receiver implementation|Description|
|---|---|
|1|---|
|2|---|
|3|---|
|4|---|
|5|---|