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

type UserService struct{}

func (s *UserService) Create(name string) (*User, error) {
	return &User{ID: 1, Name: name}, nil
}

func (s *UserService) Delete(id int) error {
	return nil
}

func (s UserService) List() ([]User, error) {
	return nil, nil
}

type Config struct {
	Host string
	Port int
}

type Reader interface {
	Read(p []byte) (n int, err error)
}

func ProcessOrder(id int) error {
	helper()
	return nil
}

func helper() {
	// internal
}
