package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

// 用于表示coordinator所处的状态
const (
	NoneStage = iota
	MapStage
	ReduceStage
)

// 表示Task所处的状态
const (
	InitTask = iota
	Running
	Ending
)

// 任务的类型
const (
	MapTask = iota
	ReduceTask
	SleepTask
)

// 任务的结构
type Task struct {
	Status    int
	Index     int
	Filename  string
	BeginTime time.Time
	TaskType  int
}

// 协调器的结构
type Coordinator struct {
	// Your definitions here.
	Status    int
	Filenames []string
	NMap      int
	NReduce   int
	Mu        sync.Mutex

	TaskQue         chan Task
	TaskQueLen      int
	RunningMap      map[int]Task
	RunningReduce   map[int]Task
	CompletedMap    map[int]Task
	CompletedReduce map[int]Task
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) GetTask(args *ExampleArgs, reply *Task) error {

	if c.Status == MapStage {
		c.Mu.Lock()
		*reply = <-c.TaskQue
		reply.Status = Running
		reply.TaskType = MapTask
		reply.BeginTime = time.Now()
		c.TaskQueLen -= 1
		c.Mu.Unlock()
		if c.TaskQueLen == 0 {
			c.Status = ReduceStage
		}
	} else if c.Status == ReduceStage {
		//先判断是否还有未运行完的Map
		return nil
	} else {
		//结束
		return nil
	}
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	c.Status = MapStage
	c.Filenames = files
	c.NMap = len(files)
	c.NReduce = nReduce
	c.TaskQue = make(chan Task, c.NMap)
	c.TaskQueLen = 0
	c.RunningMap = make(map[int]Task)
	c.CompletedMap = make(map[int]Task)
	c.RunningReduce = make(map[int]Task)
	c.CompletedReduce = make(map[int]Task)

	for i, v := range files {
		task := Task{}
		task.Status = InitTask
		task.Filename = v
		task.Index = i
		task.TaskType = MapTask
		c.TaskQue <- task
	}
	c.TaskQueLen = 0

	c.server()
	return &c
}
