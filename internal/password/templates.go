package password

type Template struct {
	Name     string   `json:"name"`
	Settings Settings `json:"settings"`
}
