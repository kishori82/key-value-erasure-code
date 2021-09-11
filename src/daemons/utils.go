package daemons

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	//"strconv"
)

func PrintHeader(title string) {
	length := len(title)
	numSpaces := 22
	leftHalf := numSpaces + int(math.Ceil(float64(length)/2))
	rightHalf := numSpaces - int(math.Ceil(float64(length)/2))
	fmt.Println("***********************************************")
	fmt.Println("*                                             *")
	fmt.Print("*")
	fmt.Printf("%*s", int(leftHalf), title)
	fmt.Printf("%*s", (int(rightHalf) + 1), " ")
	fmt.Println("*")
	fmt.Println("*                                             *")
	fmt.Println("***********************************************")
}

/**
* Print out a footer to the screen
 */
func PrintFooter() {
	fmt.Println("***********************************************")
}

func Print_configuration(proc_type uint64, ip_addrs *list.List) {

	if proc_type == 0 {
		//fmt.Printf("Process Type: %d\n", proc_type)
		fmt.Printf("Process Reader \n")
	}

	if proc_type == 1 {
		fmt.Printf("Process Writer \n")
	}
	if proc_type == 2 {
		fmt.Printf("Process Server \n")
	}

	fmt.Println("IP Addresses: ")
	for e := ip_addrs.Front(); e != nil; e = e.Next() {
		fmt.Printf("    %s\n", e.Value)
	}
}

func rand_wait_time() int64 {

	var rand_dur int64 = 1000
	/*
		if processParams.processType == 0 {
			rand_wait_time_const := func(distrib []string) int64 {
				k, err := strconv.ParseInt(distrib[1], 10, 64)
				if err != nil {
					return 100
				}
				return k
			}
			rand_dur = rand_wait_time_const(data.inter_read_wait_distribution)
		}*/

	return rand_dur
}

func NumberOfLinesInFile(filetoread string) (int, error) {
	file, err := os.Open(filetoread)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	r := io.Reader(file)

	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
	return count, nil
}

/*
package main

import (
         "fmt"
         "math/rand"
         "math"
       )
*/

type ObjectSelector interface {
	Rand() int
}

type Uniform struct {
	n int64
	u *rand.Rand
}

func (u *Uniform) Rand() int {
	return u.u.Intn(int(u.n))
}

func (z *Uniform) SetParams(n int64, seed int64) {
	z.n = n
	z.u = rand.New(rand.NewSource(seed))
}

type Zipf struct {
	n     int64
	theta float64
	u     *rand.Rand
	a     []float64
}

func (z *Zipf) SetParams(n int64, theta float64, seed int64) {
	z.n = n
	z.theta = theta
	z.u = rand.New(rand.NewSource(seed))
	z.a = make([]float64, z.n)
	z.createCumulProb()
}

func (z *Zipf) createCumulProb() {
	H := 0.0
	for i := 1; i <= int(z.n); i++ {
		H += math.Pow(1/float64(i), z.theta)
		z.a[i-1] = H
	}
	for i := 0; i < int(z.n); i++ {
		z.a[i] = z.a[i] / H
	}
}

func (z *Zipf) Rand() int {
	v := z.u.Float64()
	l := int(0)
	r := int(z.n)
	x := 0
	for l < r && x != int((l+r)/2) {
		x = int((l + r) / 2)
		if z.a[x] < v {
			l = x
		} else {
			r = x
		}
	}
	return l
}

/*
const (
	ZIPFIAN_OBJECT_PICK="zipfian"
	UNIFORM_OBJECT_PICK="uniform"
      )
*/

func createObjectPicker(n int64, distrib string) (ObjectSelector, error) {

	switch distrib {
	case ZIPFIAN_OBJECT_PICK:
		var x = new(Zipf)
		x.SetParams(n, 0.8, 99)
		return x, nil
	case UNIFORM_OBJECT_PICK:
		var y = new(Uniform)
		y.SetParams(n, 99)
		return y, nil
	}

	fmt.Println("what is the distribution :", distrib)
	return new(Uniform), errors.New("Not a valid distribution for object selection")
}

/*
func main() {
   z, _ := createObjectPicker(100, "zipfian")

   for i:=0; i< 100000; i++ {
     fmt.Println(z.Rand())
   }
}
*/
