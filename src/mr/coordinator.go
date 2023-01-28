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

type TaskQue struct {
	que []Task
	mu  sync.Mutex
}

func (t *TaskQue) Push(task Task) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.que = append(t.que, task)
}

func (t *TaskQue) Pop() Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	ret := t.que[0]
	t.que = t.que[1:]
	return ret
}

func (t *TaskQue) Len() int {
	return len(t.que)
}

// 协调器的结构
type Coordinator struct {
	// Your definitions here.
	Status    int
	Filenames []string
	NMap      int
	NReduce   int
	Mu        sync.Mutex

	InitMapQue      TaskQue
	InitReduceQue   TaskQue
	RunningMap      map[int]Task
	RunningReduce   map[int]Task
	CompletedMap    map[int]Task
	CompletedReduce map[int]Task
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) GetTask(args *ExampleArgs, reply *Task) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	if c.Status == MapStage {
		//fmt.Printf("%v %v\n", time.Now(), c.TaskQueLen)
		if c.InitMapQue.Len() == 0 {
			reply.TaskType = SleepTask
			return nil
		}
		*reply = c.InitMapQue.Pop()
		c.RunningMap[reply.Index] = *reply
		reply.Status = Running
		reply.TaskType = MapTask
		reply.BeginTime = time.Now()
	} else if c.Status == ReduceStage {
		//fmt.Printf("%v Reduce\n", time.Now())
		if c.InitReduceQue.Len() == 0 {
			reply.TaskType = SleepTask
			return nil
		}
		*reply = c.InitReduceQue.Pop()
		c.RunningReduce[reply.RIndex] = *reply
		reply.Status = Running
		reply.TaskType = ReduceTask
		reply.BeginTime = time.Now()
	} else if c.Status == IdleStage {
		//fmt.Printf("%v idle\n", time.Now())
		reply.TaskType = ExitTask
	}
	go c.CheckExcept(*reply)
	return nil
}

func (c *Coordinator) Report(info *Task, reply *ExampleReply) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	//fmt.Printf("%v report\n", time.Now())
	if info.TaskType == MapTask {

		_, ok := c.RunningMap[info.Index]
		if ok {
			delete(c.RunningMap, info.Index)
			info.Status = Ending
			c.CompletedMap[info.Index] = *info
			if len(c.CompletedMap) == c.NMap {
				//fmt.Printf("%v to reduce stage\n", time.Now())
				c.Status = ReduceStage
			}
		}

	} else if info.TaskType == ReduceTask {

		_, ok := c.RunningReduce[info.RIndex]
		if ok {
			delete(c.RunningReduce, info.RIndex)
			info.Status = Ending
			c.CompletedReduce[info.RIndex] = *info
			if len(c.CompletedReduce) == c.NReduce {
				//fmt.Printf("%v to idle stage\n", time.Now())
				c.Status = IdleStage
			}
		}
	}
	return nil
}

func (c *Coordinator) CheckExcept(task Task) {
	time.Sleep(time.Second * 10)

	c.Mu.Lock()
	defer c.Mu.Unlock()

	if task.TaskType == MapTask {
		_, ok := c.RunningMap[task.Index]
		if ok {
			delete(c.RunningMap, task.Index)
			c.InitMapQue.Push(task)
		}
	} else if task.TaskType == ReduceTask {
		_, ok := c.RunningReduce[task.RIndex]
		if ok {
			delete(c.RunningReduce, task.RIndex)
			c.InitReduceQue.Push(task)
		}
	}
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
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if len(c.CompletedMap) == c.NMap && len(c.CompletedReduce) == c.NReduce {
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
	c.InitMapQue = TaskQue{}
	c.InitReduceQue = TaskQue{}
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
		c.InitMapQue.Push(task)
	}

	for i := 0; i < nReduce; i++ {
		task := Task{}
		task.Status = InitTask
		task.Filename = ""
		task.RIndex = i
		task.NMap = c.NMap
		task.TaskType = ReduceTask
		task.NReduce = nReduce
		c.InitReduceQue.Push(task)
	}

	c.server()
	return &c
}
