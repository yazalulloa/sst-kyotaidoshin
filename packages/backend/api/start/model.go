package start

type Page struct {
	Id        string     `json:"Id"`
	Path      string     `json:"Path"`
	SubRoutes []SubRoute `json:"SubRoutes"`
}

type SubRoute struct {
	Id   string `json:"Id"`
	Path string `json:"Path"`
}
