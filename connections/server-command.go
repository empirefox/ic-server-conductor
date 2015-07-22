package connections

type ServerCommand struct {
	Name    string `json:"name"`
	Content string `json:"content,omitempty"`
}
