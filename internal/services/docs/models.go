package docs

import "time"

// Doc hold all document data
type Doc struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	FilePath    string    `json:"filePath"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// DocShare holds all document share data
type DocShare struct {
	ID        int       `json:"docShare"`
	DocID     int       `json:"docID"`
	GranteeID int       `json:"granteeID"`
	Level     string    `json:"level"`
	GrantedBy string    `json:"grantedBy"`
	GrantedAt time.Time `json:"grantedAt"`
}
