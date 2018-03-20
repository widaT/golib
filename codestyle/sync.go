package codestyle

//代码的model
import "fmt"
var actions = make (chan func())
func Run()  {
	go func() {
		for {
			select {
			case b := <- actions:
				b()
			}
		}
	}()
	r := a()
	fmt.Println(r)
}

func a () (result int){
	c := make(chan struct{})
	actions <- func() {
		defer close(c)
		//特殊的闭包返回
		result = 1
	}
	<-c
	return result
}