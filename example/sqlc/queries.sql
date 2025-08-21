-- name: GetUser :one
SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListUsers :many
SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: SearchUsers :many
SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
FROM users
WHERE deleted_at IS NULL /* sqld:where */
ORDER BY created_at DESC, id DESC /* sqld:orderby */ /* sqld:cursor */ /* sqld:limit */;

-- name: CreateUser :one
INSERT INTO users (name, email, age, status, role, country, verified)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at;

-- name: UpdateUser :one
UPDATE users
SET name = $2, email = $3, age = $4, status = $5, role = $6, country = $7, verified = $8, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at;

-- name: DeleteUser :exec
UPDATE users SET deleted_at = NOW() WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: ListUsersByStatus :many
SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
FROM users
WHERE status = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: SearchUsersByStatus :many
SELECT id, name, email, age, status, role, country, verified, created_at, updated_at, deleted_at
FROM users
WHERE status = $1 AND deleted_at IS NULL /* sqld:where */
ORDER BY created_at DESC, id DESC /* sqld:orderby */ /* sqld:cursor */ /* sqld:limit */;

-- name: GetPost :one
SELECT id, user_id, title, content, published, category, tags, created_at, updated_at
FROM posts
WHERE id = $1;

-- name: ListPosts :many
SELECT id, user_id, title, content, published, category, tags, created_at, updated_at
FROM posts
ORDER BY created_at DESC;

-- name: ListPostsByUser :many
SELECT id, user_id, title, content, published, category, tags, created_at, updated_at
FROM posts
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: CreatePost :one
INSERT INTO posts (user_id, title, content, published, category, tags)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, title, content, published, category, tags, created_at, updated_at;

-- name: UpdatePost :one
UPDATE posts
SET title = $2, content = $3, published = $4, category = $5, tags = $6, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, title, content, published, category, tags, created_at, updated_at;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;