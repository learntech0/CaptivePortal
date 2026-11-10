package backend

type PayRequest struct {
	Phone    string  `json:"phone"`
	Amount   float64 `json:"amount"`
	Mac      string  `json:"mac"`
	RouterIp string  `json:"router_ip"`
}