package auth

type Permission string

const (
	// Auth Permissions
	PermUserCreate Permission = "user:create"
	PermUserBan    Permission = "user:ban"

	// File Upload Permissions
	PermFileCreate Permission = "file:upload"
	PermFileDelete Permission = "file:delete"

	// AI Permissions
	PermAIUse Permission = "ai:use"

	// Blog Permissions
	PermBlogCreate  Permission = "blog:create"
	PermBlogPublish Permission = "blog:publish"
)

var AllPermissions = []Permission{
	PermUserCreate, PermUserBan, PermFileCreate, PermFileDelete, PermAIUse, PermBlogCreate, PermBlogPublish,
}
