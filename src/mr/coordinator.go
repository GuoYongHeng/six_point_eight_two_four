package mr

import (
	"fmt"
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
	IdleStage
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
	ExitTask
)

// 任务的结构
type Task struct {
	Status    int
	Index     int //map index
	RIndex    int //reduce index
	NReduce   int
	NMap      int
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
	ReduceQue       chan Task
	ReduceQueLen    int
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
		fmt.Printf("%v %v\n", time.Now(), c.TaskQueLen)
		*reply = <-c.TaskQue
		reply.Status = Running
		reply.TaskType = MapTask
		reply.BeginTime = time.Now()
		c.TaskQueLen -= 1
		c.RunningMap[reply.Index] = *reply
		c.Mu.Unlock()
		if c.TaskQueLen == 0 {
			c.Status = ReduceStage
		}
	} else if c.Status == ReduceStage {
		c.Mu.Lock()
		if len(c.CompletedMap) == c.NMap {
			fmt.Printf("%v Reduce\n", time.Now())
			*reply = <-c.ReduceQue
			reply.Status = Running
			reply.TaskType = ReduceTask
			reply.BeginTime = time.Now()
			c.ReduceQueLen -= 1
			c.RunningReduce[reply.RIndex] = *reply
			if c.ReduceQueLen == 0 {
				c.Status = IdleStage
			}
		} else {
			fmt.Printf("%v map running\n", time.Now())
			reply.TaskType = SleepTask
		}
		c.Mu.Unlock()
	} else if c.Status == IdleStage {
		fmt.Printf("%v idle\n", time.Now())
		reply.TaskType = ExitTask
	}
	return nil
}

func (c *Coordinator) Report(info *Task, reply *ExampleReply) error {
	fmt.Printf("%v report\n", time.Now())
	if info.TaskType == MapTask {

		_, ok := c.RunningMap[info.Index]
		if ok {
			delete(c.RunningMap, info.Index)
			info.Status = Ending
			c.CompletedMap[info.Index] = *info
		}

	} else if info.TaskType == ReduceTask {

		_, ok := c.RunningReduce[info.RIndex]
		if ok {
			delete(c.RunningReduce, info.RIndex)
			info.Status = Ending
			c.CompletedReduce[info.RIndex] = *info
		} else {
			c.Status = NoneStage
		}
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
	if c.Status == NoneStage {
		return true
	}
	// Your code here.

	return false
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
	c.ReduceQue = make(chan Task, nReduce)
	c.ReduceQueLen = 0
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
		task.NReduce = c.NReduce
		task.NMap = c.NMap
		c.TaskQue <- task
	}
	c.TaskQueLen = c.NMap

	for i := 0; i < nReduce; i++ {
		task := Task{}
		task.Status = InitTask
		task.Filename = ""
		task.RIndex = i
		task.NMap = c.NMap
		task.TaskType = ReduceTask
		task.NReduce = nReduce
		c.ReduceQue <- task
	}
	c.ReduceQueLen = c.NReduce

	c.server()
	return &c
}
