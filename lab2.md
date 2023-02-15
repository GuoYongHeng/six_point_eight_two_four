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
* 按照论文中的图2，在这里你应该关注发送和接收RequestVote、与选举相关的服务器规则和与leader选举相关的状态
* 在raft.go中的Raft结构中为leader选举添加图2的状态。你也需要定义一个结构去保存和每个log entry相关的信息
* 填充`RequestVoteArgs`和`RequestVoteReply`结构。修改`Make()`去创建一个后台协程，该协程在一段时间内没有收到其他对等节点的消息时将会通过发送`RequestVote`来定期开始leader选举。通过这种方式peer将会知道谁是leader，如果已经存在leader的话，或者它自己会变成leader。实现`RequestVote()`RPC函数以便服务器互相投票
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

#### Leader选举大致流程
* 服务器启动时，全都是Follower
* Follower超过一定时间没有收到心跳，就会成为Candidate，发起选举
* 开始选举时，Follower首先增加Term，然后转换为Candidate，然后并行向集群其他节点发送请求投票的RPC
* Candidate会保持当前状态直到以下三件事情之一发生
  * 当前节点赢得选举成为Leader
  * 别的节点成为Leader
  * 一段时间后没有节点成为Leader
* 一旦Candidate赢得选举成为Leader，便会向其余节点发送心跳，阻止其余节点发起新的选举
* Candidate等待投票时，可能会受到其他服务器的AppendEntries，如果受到信息中的Term大于等于当前节点Term，则当前节点承认Leader并且状态切换到Follower，否则，拒绝请求
* 如果某轮选举有多个Follower同时成为Candidate，则可能导致无法选出Leader，此时每个Candidate都会超时，然后每个节点会增加Term来开始新一轮选举
* Raft算法使用随机选举超时时间的方法来减少上一条情况的发生。


### Part 2B:log

#### 任务
实现leader和follower的添加新的日志条目的代码，以便`go test -run 2B`测试的通过

#### 提示
* 运行`git pull`以获得最新的lab software
* 你的第一个目标应该是通过`TestBasicAgree2B()`。通过实现`Start()`开始，然后编写代码通过`AppendEntries` RPCs来发送和接收新的日志条目，如图2所示。在每个对等端的`applyCh`上发送每个新的已提交的条目。
* 你需要实现选举限制(论文中的5.4.1节)
* 在早期的Lab 2B测试中无法达成一致的一种方法是即使领导者还活着，也要进行重复选举。寻找选举定时器管理中的bug，或者在赢得选举后不要立即发送心跳。
* 你的代码可能包含重复检查某些事件的循环。不要让这些循环连续不间断的执行，因为那将会减慢你的实现的速度，以至于无法通过测试。使用Go的<u>condition variables</u>，或者在每次循环迭代中插入`time.Sleep(10 * time.Millisecond`
* 为你之后的lab帮个忙，编写干净清晰的代码。如果需要一些想法，可以重新访问[Guidance page](https://pdos.csail.mit.edu/6.824/labs/guidance.html)，其中包含怎么开发和调试代码。
* 如果你没有通过测试，查看config.go和test_test.go的代码，可以更好地理解测试正在测试什么。config.go中也说明了tester怎么使用Raft API

如果运行的太慢，你的代码可能无法通过即将到来的实验的测试。你可以使用time命令检查你的解决方案的真实耗时和cpu时间。典型的输出如下：
```shell
$ time go test -run 2B
Test (2B): basic agreement ...
  ... Passed --   0.9  3   16    4572    3
Test (2B): RPC byte count ...
  ... Passed --   1.7  3   48  114536   11
Test (2B): agreement after follower reconnects ...
  ... Passed --   3.6  3   78   22131    7
Test (2B): no agreement if too many followers disconnect ...
  ... Passed --   3.8  5  172   40935    3
Test (2B): concurrent Start()s ...
  ... Passed --   1.1  3   24    7379    6
Test (2B): rejoin of partitioned leader ...
  ... Passed --   5.1  3  152   37021    4
Test (2B): leader backs up quickly over incorrect follower logs ...
  ... Passed --  17.2  5 2080 1587388  102
Test (2B): RPC counts aren't too high ...
  ... Passed --   2.2  3   60   20119   12
PASS
ok  	6.5840/raft	35.557s

real	0m35.899s
user	0m2.556s
sys	0m1.458s
$
```
"ok 6.5840/raft 35.557s"意味着Go测量的2B测试所用的实际时间(wall-clock)为35.557秒，"user 0m2.556s"表示代码消耗了2.556秒的CPU时间，或者实际执行指令花费的时间（而不是等待或休眠）。如果你的解决方案在2B测试中花费的时间超过了1分钟，或者CPU时间超过5秒钟，你以后可能会遇到麻烦。查看时间花费在了哪里，比如休眠或者等待rpc超时花费的时间，未休眠时的循环或者等待条件变量或者channel传递消息所花费的时间，或者发送大量rpc花费的时间。

#### 日志复制大致流程
* 论文图2中指明的日志接受者的一些规则
  * 如果Leader Term小于接收节点Term，则返回false
  * 如果接收节点中没有包含这样一条日志，即prevLogIndex位置上的日志的Term可以和prevLogTerm匹配上的日志，则返回false
  * 如果已经存在的日志和新日志发生了冲突（同样的索引但是不同的Term），那么就删除当前日志及它后边的所有日志
  * 追加日志中尚未存在的任何新日志
  * 如果leaderCommit>commitIndex，则设置commitIndex=min(leaderCommit,上一个新日志的索引)
* 论文5.4节涉及到的规则
  * Raft 通过比较两份日志中最后一条日志条目的索引值和任期号定义谁的日志比较新。如果两份日志最后的条目的任期号不同，那么任期号大的日志更加新。如果两份日志最后的条目任期号相同，那么日志比较长的那个就更加新

### Part 2C:persistence
如果一个基于Raft的服务器重启，它应该恢复服务到它离线的位置。这要求Raft可以保持重启后仍然存在持久的状态。论文中图2提到了哪些状态应该被持久化。

一个真正的实现应该在Raft的持久状态改变时将它写入磁盘，并且服务重启后应该从磁盘读取状态。你的实现中不需要使用磁盘，相反，你的代码中应该save和restore持久状态到Persister对象（见persister.go）。无论是谁调用`Raft.Make()`到会提供一个Persister对象，该对象最初持有Raft的最近的持久化状态（如果存在）。Raft应该用Persister对象初始化自己，并且应该在每次状态改变时使用Persister对象来保存他的持久状态。使用Persister的ReadRaftState()和Save函数。
#### 任务
通过添加代码去保存和恢复持久状态来完成raft.go中的`persist()`和`readPersist()`。你需要去编码（或序列化）状态为字节数组，以便将其传递给Persister。使用labgob编码器，看`persist()`和`readPersist()`种的注释。labgob类似go的gob编码器，但是如果你尝试编码小字段名的结构体，它将打印错误信息。现在，将`nil`作为第二个参数传递给`persist.Save()`。在你的实现更改持久状态的位置插入对`persist()`的调用。完成此操作后，如果你的其余的实现是正确的，你将会通过所有的2C测试。

你可能需要一次通过多个条目备份nextIndex的优化。查看论文中第七页底部和第八页顶部的内容（灰线标记）。论文中对细节的描述含糊不清，你需要去填补这部分空白。一种可能是拒绝信息包括如下：
```
XTerm:  term in the conflicting entry (if any)
XIndex: index of first entry with that term (if any)
XLen:   log length
```
然后Leader的逻辑可能是这样：
```
Case 1: leader doesn't have XTerm:
  nextIndex = XIndex
Case 2: leader has XTerm:
  nextIndex = leader's last entry for XTerm
Case 3: follower's log is too short:
  nextIndex = XLen
```

#### 提示
* 运行`git pull`去获取最新的代码
* 2C的测试要求比2A和2B的要求更高，失败可能是因为你的2A或2B的代码有问题

你的代码应该通过所有的2C的测试（如下所示），以及2A和2B的测试。
```
$ go test -run 2C
Test (2C): basic persistence ...
  ... Passed --   5.0  3   86   22849    6
Test (2C): more persistence ...
  ... Passed --  17.6  5  952  218854   16
Test (2C): partitioned leader and one follower crash, leader restarts ...
  ... Passed --   2.0  3   34    8937    4
Test (2C): Figure 8 ...
  ... Passed --  31.2  5  580  130675   32
Test (2C): unreliable agreement ...
  ... Passed --   1.7  5 1044  366392  246
Test (2C): Figure 8 (unreliable) ...
  ... Passed --  33.6  5 10700 33695245  308
Test (2C): churn ...
  ... Passed --  16.1  5 8864 44771259 1544
Test (2C): unreliable churn ...
  ... Passed --  16.5  5 4220 6414632  906
PASS
ok  	6.5840/raft	123.564s
$
```
在提交之前运行多次测试并检查每次运行打印的PASS是个好主意。
```shell
$ for i in {0..10}; do go test; done
```

### Part 2D:log compaction
按照现在的情况，重启服务器会重放完整的Raft log去恢复它的状态。然而，对于一个长时间运行的服务来说，永远记住完整的Raft log是不现实的。相反，你将会修改Raft与一个这样的服务来合作，该服务会不时的持久化存储他们状态的快照，此时Raft会丢弃快照之前的日志条目。结果是更少量的持久化数据和更快的重启速度。然而，现在可能跟随者落后太多以至于leader丢弃了跟随者需要赶上的日志条目。leader必须发送一个快照加上快照之后的日志条目。论文中的第7节概述了该方案。你需要去设计方案细节。

你可能会发现参考[Raft交互图](https://pdos.csail.mit.edu/6.824/notes/raft_diagram.pdf)有助于理解复制服务和Raft的通信方式。

你的服务必须提供以下函数，服务可以使用其状态的序列化快照调用该函数。
```go
Snapshot(index int, snapshot []byte)
```

在实验2D中，测试周期的调用Snapshot()。在实验3中，你将会写一个调用Snapshot的key/value服务器。snapshot将会包含完成的key/value对表格。服务层面在每个对等点调用Snapshot()（不仅仅是leader）。

index参数指明快照中反应的最高的log entry。Raft应该丢弃在他之前的log entries。你需要去修改你的Raft代码以在仅存储日志尾部的同时进行操作。

你需要去实现论文中讨论的InstallSnapshot RPC，他允许Raft leader去告诉落后的Raft对端节点用snapshot去替换他的状态。你可能需要仔细考虑InstallSnapshot应如何与图2中的状态和规则交互。

当跟随者的Raft代码接收到一个InstallSnapshot RPC，他可以使用applyCh去发送snapshot到ApplyMsg中的服务。ApplyMsg结构体早就定义了你需要的字段（包括测试期望的字段）。请注意这些快照只会推进服务状态，不会导致它向后移动。

如果服务器崩溃，他必须从持久化数据中重新启动。你的Raft应该保留Raft状态和响应的快照。使用persister.Save()的第二个参数去保存快照。如果没有快照，则第二个参数传送nil。

当服务器重启的使用，应用层读取持久化的快照并且恢复它保存的状态。

#### 任务
实现Snapshot()和InstallSnapshot RPC，以及更改Raft来支持这些（例如，修剪日志的操作）。当你的解决方案完整的通过2D测试（以及之前所有的实验2测试），你的解决方案就完成了。

#### 提示
* git pull确保你有最新的代码
* 修改你的代码的一个好的开始时以便它能够仅从某个索引X开始的日志部分。最初，你可以设置X为0并运行2B/2C测试。然后让Snapshot(index)丢弃index之前的的日志，然后设置X为index。如果一切顺利，你现在应该可以通过2D的第一个测试。
* 你不能存储日志到Go切片中，也无法使用Go切片索引与Raft日志索引互换使用。你需要以一种考虑日志丢弃部分的方式对切片进行索引。
* 下一步：让leader发送InstallSnapshot RPC，如果它没有更新追随者所需要的条目。
* 在单个InstallSnapshot RPC发送整个snapshot。不要实现图13中的用于拆分快照的offset机制。
* Raft必须允许Go垃圾收集器释放和重新使用内存的方式丢弃旧的日志条目，这要求对丢弃的日志条目没有可访问的引用（指针）。
* 即使日志被修剪，你的实现仍然需要在AppendEntries RPC中的新条目之前正确发送条目的term和index。这可能需要保存并引用最新的快照lastIncludedTerm/lastIncludedIndex（考虑是否应该保留）。
* 没有-race的全套实验2测试（2A+2B+2C+2D）消耗的合理时间是6分钟真实时间和1分钟CPU时间。当使用-race运行的时候，大约是10分钟的真实时间和2分钟的CPU时间。

你应该通过所有的2D测试（如下所示），以及2A，2B和2C的测试。
```shell
$ go test -run 2D
Test (2D): snapshots basic ...
  ... Passed --  11.6  3  176   61716  192
Test (2D): install snapshots (disconnect) ...
  ... Passed --  64.2  3  878  320610  336
Test (2D): install snapshots (disconnect+unreliable) ...
  ... Passed --  81.1  3 1059  375850  341
Test (2D): install snapshots (crash) ...
  ... Passed --  53.5  3  601  256638  339
Test (2D): install snapshots (unreliable+crash) ...
  ... Passed --  63.5  3  687  288294  336
Test (2D): crash and restart all servers ...
  ... Passed --  19.5  3  268   81352   58
PASS
ok      6.5840/raft      293.456s
```











## State
|Persistent state on all servers|Description|
|---|---|
|currentTerm|---|
|votedFor|---|
|log[]|（初始索引为1）|

|Volatile state on all servers|Description|
|---|---|
|commitIndex|已知被提交的最新的log entry的索引（初始值为0，单调递增）|
|lastApplied|已经被应用到状态机的最新的log entry的索引（初始值为0，单调递增）|

|Volatile state on leaders|Description|
|---|---|
|nextIndex[]|nextIndex的初始值为Leader的最后一个log entry的索引+1|
|matchIndex[]|---|

## AppendEntries RPC
|Arguments|Description|
|---|---|
|term|---|
|leaderId|---|
|prevLogIndex|nextIndex[i]-1所指位置的日志的索引|
|prevLogTerm|上边那个日志所属的Term|
|entries[]|---|
|leaderCommit|Leader的commitIndex|

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