package testdata

const (
	Version = "2.0.0"
	Dip     = true
)

var GlobalConfig = "default"

type User struct {
	ID   int
	Name string
	Age  int
}

type UserService struct {
	db string
	ns string
}

func (s *UserService) Create(name string) (*User, error) {
	return &User{ID: 1, Name: name + "-"}, nil
}

func (s UserService) List() ([]*User, error) {
	return nil, nil
}

func (s *UserService) Delete(id int) error {
	return nil
}

type Reader interface {
	Read(p []byte) (n int, err error)
}

func ProcessOrder(id int) error {
	return nil
}

func helper() string {
	return "internal"
}
