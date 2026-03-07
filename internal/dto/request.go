package dto

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type GoogleAuthRequest struct {
	Credential string `json:"credential" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type CreateLinkRequest struct {
	OriginalURL string `json:"original_url" binding:"required,url"`
	Title       string `json:"title"`
}

type UpdateLinkRequest struct {
	Title    *string `json:"title"`
	IsActive *bool   `json:"is_active"`
}

type CreateSourceRequest struct {
	SourceName string `json:"source_name" binding:"required,min=1,max=100"`
}

type CreateGroupRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

type UpdateGroupRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

type AssignGroupLinkRequest struct {
	LinkID string `json:"link_id" binding:"required,uuid"`
}

type BatchSourceRequest struct {
	SourceName string `json:"source_name" binding:"required,min=1,max=100"`
	ScopeType  string `json:"scope_type" binding:"required,oneof=all group"`
	ScopeID    string `json:"scope_id"`
}
