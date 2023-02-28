## 相关资料

### lab4介绍
你可以根据你自己的想法做一个[最终项目](https://pdos.csail.mit.edu/6.824/project.html)，也可以做这个lab。

在本实验中，你将构建一个键/值存储系统，该系统对一组副本上的键进行分片或分区。一个分片是键值对的子集。例如，所有以a开头的键可能是一个分片，所有以b开头的键可能是另一个分片，等等。分片的原因是性能。每个副本组仅处理少量分片的get和put，并且这些组可以并行操作，因此总系统吞吐量（get和put每单位时间内）与组数成比例增加。

你的分片键值存储将会有两个主要组件。首先，一组副本组。每个副本组负责分片的一个子集。副本由几个服务器组成，这些服务器使用Raft来复制组的分片。第二个组件是分片控制器。分片控制器决定哪个副本组应该为哪个分片服务，这些信息成为配置。配置随时间变化。客户端咨询分片控制器所要服务的分片。整个系统只有一个分片控制器，使用Raft作为容错服务实现。

一个分片存储系统必须能够在副本组之间转移分片。一个原因是一些组可能比其他组的负载高，因此需要移动分片来均衡负载。另一个原因是副本组可能加入和离开系统：可能会添加新的副本组增加容量，或者现有的副本组可能因为维修或到期而下线。

在这个实验中主要的挑战将是处理重新配置——分片分配到组的变化。在单个副本组内，当一个和客户端Put/Append/Get请求相关的重新配置发生时，所有的组成员都必须同意。例如，Put可能和重新配置同时到达，导致副本组不再对持有的Put的key对应的分片负责。组中的所有副本必须对Put是发生在重新配置之前还是之达成一致。如果Put是在重新配置之前达到，Put应该生效并且新的分片所有者将会看到他的影响。如果Put在重新配置之后到达，Put将不会产生影响并且客户端必行在新的所有者任期内进行重试。推荐的方法是让每个副本组使用Raft不仅记录Put/Append/Get的顺序，而且还记录重新配置的顺序。你将需要确保在任何时候最多有一个副本组为每个分片的请求提供服务。

重新配置也需要在副本组之间交互。例如，在配置10中，组G1可能负责分片S1.在配置11中，组G2可能负责分片S1.在从10到11的重新配置期间，G1和G2必须使用RPC将分片S1的内容（键值对）从G1移动到G2。

**注意：RPC只能被用于客户端和服务端之间的交互。例如，你的服务器的不同实例间不允许使用
rpc、共享变量或文件**

**注意：本实验使用"configuration"来指代分片到副本组的分配。这与Raft集群成员变化不同。你不必实施Raft集群成员更改**

本实验的通用架构（一个配置服务和一组副本组）准寻与平面数据中心存储、BigTable、Spanner、FAWN、Apache HBase、Rosebud、Spinnaker等相同的通用模式。不过，这些系统在细节上与本实验有许多不同，而且通常也更复杂和强大。例如，本实验不会再每个raft组中涉及对等点集，它的数据和查询模型非常简单，分片的切换很慢并且不允许客户端并发访问。

**你的lab4分片服务器，lab分片空腹之气和lab3必须使用同样的Raft实现。我们将重新运行lab2和lab3的测试作为lab4评分的一部分，你在旧的测试中的分数将会计入你的lab4的总成绩。这些测试在你的lab4总成绩中占10分。**

#### 开始
**重点：git pull获得最新的代码**

我们在src/shardctrler和src/shardkv中提供了框架代码和测试。

要启动并运行，请运行下面的命令：
```shell
$ cd ~/6.5840
$ git pull
...
$ cd src/shardctrler
$ go test
--- FAIL: TestBasic (0.00s)
        test_test.go:11: wanted 1 groups, got 0
FAIL
exit status 1
FAIL    shardctrler     0.008s
$
```
完成后，你的实现应该通过src/shardctrler目录中的所有测试，以及src/shardkv中的所有测试。

#### A部分：分片控制器
首先，你将在shardctrler/server.go和client.go中实现分片控制器。完成后，你应该通过shardctrler目录中的所有测试。
```shell
$ cd ~/6.5840/src/shardctrler
$ go test
Test: Basic leave/join ...
  ... Passed
Test: Historical queries ...
  ... Passed
Test: Move ...
  ... Passed
Test: Concurrent leave/join ...
  ... Passed
Test: Minimal transfers after joins ...
  ... Passed
Test: Minimal transfers after leaves ...
  ... Passed
Test: Multi-group join/leave ...
  ... Passed
Test: Concurrent multi leave/join ...
  ... Passed
Test: Minimal transfers after multijoins ...
  ... Passed
Test: Minimal transfers after multileaves ...
  ... Passed
Test: Check Same config on servers ...
  ... Passed
PASS
ok  	6.5840/shardctrler	5.863s
$
```
shardctrler管理一系列编号的配置。每一个配置描述一组副本和分片到副本组的分配。每当此分配需要更改时，分片控制器都会创建新的分配和新的配置。剪枝客户端和服务器想要知道当前（或者过去）配置时联系shardctrler。

你的实现必须支持shardctrler/common.go中描述的RPC接口，它由Join，Leave,Move和Query RPC组成。这些RPC旨在允许一个管理员（和测试）去控制shardctrler：添加新的副本组，移除副本组和在副本组之间移动分片。

管理员使用Join RPC增加新的副本组。它的参数是一组从唯一的非零副本组标识符（GID）到服务器名字列表的映射。shardctrler应该通过创建一个包含新的副本组的新配置来做出反应。新配置应尽可能的将分片均匀的分配到副本组集合中，并且应该尽可能的移动更少的分片来达到该目标。如果GID不是当前配置的一部分，shardctrler应该允许重新使用GID（即GID应该被允许Join，然后Leave，然后再Join）。

Leave RPC的参数是之前已经加入组的GID列表。shardctrler应该创建一个不包含这些组的新配置，并将这些组的分片分配到其与组。新的配置应该在组之间尽可能均匀的划分分片，并应该尽可能少的移动分片来达到目标。

Move RPC的参数是分片编号和GID。shardctrler应该创建一个新的配置，其中分片被分配到组。Move的目的是让我们测试你的软件。Join或Leave跟在一个Move知乎就想没有去Move。因为Join和Leave会重新平衡。

Query RPC的参数是一个配置号。shardctrler回复具有该编号的配置。如果编号是-1或者比最大的已知配置编号大，shardctrler应该恢复最新的配置。Query(-1)的结果反映shardctrler在收到Query(-1)前完成处理的每一个Join，Leave或Move。

第一个配置应该编号为0。他不应该包含任何组，并且所有的分片都应该被分配到GID 0（一个无效的GID）。下一个配置（为响应Join RPC而创建）应该编号为1。通常分片明显会比组要多（即，每组将会服务多个分片），以便可以以相当精细的粒度转移负载。

#### 任务：
你的任务是在shardctrler/目录下的client.go和server.go中实现上面指定的接口。你的shardctrler必须是容错的，使用来自lab 2/3的raft库。注意我们在对lab 4进行评分时将会重新运行lab 2和lab 3的测试，以确保你没有引入bug到你的raft实现中。当你通过shardctrler中的所有测试时，你就完成了此任务。

#### 提示
* 从kvraft的精简副本开始
* 你应该为分片控制器实现RPC的重复客户端请求检测。shardctrler并不会对此进行测试，但是shardkv测试稍后将会在不可靠的网络上使用你的shardctrler。如果你的shardctrler不能过滤重复的RPC，那你可能无法通过shardkv的测试。
* 你的状态机中执行分片重新平衡的代码需要是确定型的。在go中，map迭代顺序是不确定的。
* go的map是引用。如果你将一个map类型的变量分配给另一个变量，则两个变量引用的是同一个map。因此如果你想基于之前的Config创建一个新的Config，你需要去闯进一个新的map对象（使用make()）并且单独复制键和值。
* go的race探测器（go test -race可能会帮助你发现bug。

#### B部分：分片的key/value服务器
**重点：执行git pull获取最新代码**
