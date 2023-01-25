package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}
type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	for {
		task := Task{}
		CallGetTask(&task)
		if task.TaskType == MapTask {
			//fmt.Printf("%v map\n", time.Now())
			MapWorker(mapf, &task)
		} else if task.TaskType == ReduceTask {
			//fmt.Printf("%v reduce\n", time.Now())
			ReduceWorker(reducef, &task)
		} else if task.TaskType == SleepTask {
			//fmt.Printf("%v sleep\n", time.Now())
			time.Sleep(time.Second * 5)
		} else {
			//fmt.Printf("%v end\n", time.Now())
			break
		}
	}

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

func MapWorker(mapf func(string, string) []KeyValue, task *Task) {
	file, err := os.Open(task.Filename)
	if err != nil {
		log.Fatalf("cannot open %v", task.Filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", task.Filename)
	}
	file.Close()
	kva := mapf(task.Filename, string(content))
	intermediate := []KeyValue{}
	intermediate = append(intermediate, kva...)
	interFile := make(map[int][]KeyValue)
	//划分成NReduce组
	for _, kv := range intermediate {
		idx := ihash(kv.Key) % task.NReduce
		interFile[idx] = append(interFile[idx], kv)
	}
	//生成临时文件然后存入临时文件，然后写入成功后再改成正常文件
	//这里要写入到json文件
	filenames := make(map[int]*os.File)
	for idx, kvs := range interFile {
		tmpofile, _ := ioutil.TempFile("", "mr-*")
		enc := json.NewEncoder(tmpofile)
		for _, v := range kvs {
			enc.Encode(&v)
			//fmt.Fprintf(tmpofile, "%v %v\n", v.Key, v.Value)
		}
		tmpofile.Close()
		filenames[idx] = tmpofile
	}

	for i, f := range filenames {
		os.Rename(f.Name(), "mr-"+strconv.Itoa(task.Index)+"-"+strconv.Itoa(i))
	}
	//fmt.Printf("%v map call report\n", time.Now())
	CallReport(task)
}

func ReduceWorker(reducef func(string, []string) string, task *Task) {
	intermediate := []KeyValue{}
	for i := 0; i < task.NMap; i++ {
		inName := "mr-" + strconv.Itoa(i) + "-" + strconv.Itoa(task.RIndex)
		inFile, _ := os.Open(inName)
		dec := json.NewDecoder(inFile)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
		inFile.Close()
	}

	sort.Sort(ByKey(intermediate))
	tmpofile, _ := ioutil.TempFile("", "mr-out-*")
	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)
		fmt.Fprintf(tmpofile, "%v %v\n", intermediate[i].Key, output)
		i = j
	}
	tmpofile.Close()
	os.Rename(tmpofile.Name(), "mr-out-"+strconv.Itoa(task.RIndex))
	//fmt.Printf("%v reduce call report\n", time.Now())
	CallReport(task)
}

func CallGetTask(rsp *Task) {
	args := ExampleArgs{}
	call("Coordinator.GetTask", &args, rsp)
}

func CallReport(task *Task) {
	reply := ExampleReply{}
	call("Coordinator.Report", task, &reply)
}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
