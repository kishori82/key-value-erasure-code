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
	for l < r &&  x != int((l + r) / 2) {
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
	return nil, errors.New("Not a valid distribution for object selection")
}

/*
func main() {
   z, _ := createObjectPicker(100, "zipfian") 

   for i:=0; i< 100000; i++ {
     fmt.Println(z.Rand())
   }
}
*/
