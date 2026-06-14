package dashboard

type StatsResponse struct {
	TotalContacts       int `json:"total_contacts"`
	ActiveConversations int `json:"active_conversations"`
	ResolvedToday       int `json:"resolved_today"`
	UnassignedCount     int `json:"unassigned_count"`
}

type ChartDataPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type ChartResponse struct {
	Data []ChartDataPoint `json:"data"`
}
