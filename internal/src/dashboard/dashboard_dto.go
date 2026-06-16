package dashboard

import "encoding/json"

type StatsResponse struct {
	Summary       SummaryResponse       `json:"summary"`
	Conversations ConversationsResponse `json:"conversations"`
	Contacts      ContactsResponse      `json:"contacts"`
	Messages      MessagesResponse      `json:"messages"`
	Campaigns     CampaignsResponse     `json:"campaigns"`
	Agents        []AgentWorkload       `json:"agents"`
}

type SummaryResponse struct {
	TotalContacts            int `json:"total_contacts"`
	NewContacts              int `json:"new_contacts"`
	TotalConversations       int `json:"total_conversations"`
	ActiveConversations      int `json:"active_conversations"`
	NewConversations         int `json:"new_conversations"`
	ResolvedToday            int `json:"resolved_today"`
	TotalUnreadConversations int `json:"total_unread_conversations"`
	TotalUnreadMessages      int `json:"total_unread_messages"`
	TotalMessages            int `json:"total_messages"`
	MessagesToday            int `json:"messages_today"`
	TotalCampaigns           int `json:"total_campaigns"`
	ConnectedWhatsappDevices int `json:"connected_whatsapp_devices"`
}

type ConversationsByStatus struct {
	Unassigned int `json:"unassigned"`
	Assigned   int `json:"assigned"`
	Resolved   int `json:"resolved"`
}

type ConversationsByChannel struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Total       int    `json:"total"`
	Unread      int    `json:"unread"`
}

type ChannelBrief struct {
	ID   int              `json:"id"`
	Name string           `json:"name"`
	Logo json.RawMessage  `json:"logo"`
}

type AgentBrief struct {
	ID        int              `json:"id"`
	FirstName string           `json:"first_name"`
	LastName  *string          `json:"last_name"`
	Avatar    json.RawMessage  `json:"avatar"`
}

type ContactBrief struct {
	ID        int              `json:"id"`
	FirstName string           `json:"first_name"`
	LastName  *string          `json:"last_name"`
	Avatar    json.RawMessage  `json:"avatar"`
}

type RecentConversation struct {
	ID           int                `json:"id"`
	Status       string             `json:"status"`
	UnreadCount  int                `json:"unread_count"`
	LastActivity *string            `json:"last_activity"`
	LastMessage  json.RawMessage    `json:"last_message"`
	Channel      *ChannelBrief      `json:"channel"`
	Agent        *AgentBrief        `json:"agent"`
	Contact      *ContactBrief      `json:"contact"`
}

type ConversationsResponse struct {
	ByStatus  ConversationsByStatus    `json:"by_status"`
	ByChannel []ConversationsByChannel `json:"by_channel"`
	Recent    []RecentConversation     `json:"recent"`
	Trend     []ChartDataPoint         `json:"trend"`
}

type ContactsByStatus struct {
	Individual  int `json:"individual"`
	Institution int `json:"institution"`
}

type ContactsResponse struct {
	ByStatus    ContactsByStatus `json:"by_status"`
	NewContacts int              `json:"new_contacts"`
}

type MessagesBySenderType struct {
	Contact int `json:"contact"`
	User    int `json:"user"`
}

type MessagesByDate struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type MessagesResponse struct {
	Total        int                 `json:"total"`
	Today        int                 `json:"today"`
	BySenderType MessagesBySenderType `json:"by_sender_type"`
	ByDate       []MessagesByDate    `json:"by_date"`
}

type CampaignsResponse struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
}

type AgentWorkload struct {
	ID                   int              `json:"id"`
	FirstName            string           `json:"first_name"`
	LastName             *string          `json:"last_name"`
	Avatar               json.RawMessage  `json:"avatar"`
	OngoingConversations int              `json:"ongoing_conversations"`
}

type ChartDataPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type ChartResponse struct {
	Data []ChartDataPoint `json:"data"`
}
