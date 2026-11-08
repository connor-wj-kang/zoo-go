package store

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"zoo-go/internal/domain"
)

func (s *Store) communitySummary(viewerID string, limit int) (domain.CommunitySummary, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM community_posts`).Scan(&total); err != nil {
		return domain.CommunitySummary{}, err
	}
	feed, err := s.CommunityPosts(viewerID, "", limit)
	if err != nil {
		return domain.CommunitySummary{}, err
	}
	return domain.CommunitySummary{Total: total, Latest: feed.Items}, nil
}

func (s *Store) CommunityPosts(viewerID, cursor string, limit int) (domain.CommunityFeed, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	args := make([]any, 0, 4)
	where := ""
	if cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return domain.CommunityFeed{}, validation("分页游标无效")
		}
		parts := strings.SplitN(string(decoded), "|", 2)
		if len(parts) != 2 {
			return domain.CommunityFeed{}, validation("分页游标无效")
		}
		where = "WHERE (p.created_at < ? OR (p.created_at = ? AND p.id < ?))"
		args = append(args, parts[0], parts[0], parts[1])
	}
	args = append(args, limit+1)
	rows, err := s.db.Query(`
		SELECT p.id, p.author_id, u.name, u.avatar, u.owner_verified, u.student_verified,
		       p.content, p.images_json, p.created_at
		FROM community_posts p JOIN users u ON u.id = p.author_id
		`+where+`
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return domain.CommunityFeed{}, err
	}
	defer rows.Close()
	items := []domain.CommunityPost{}
	for rows.Next() {
		var post domain.CommunityPost
		var imagesJSON, createdAt string
		var ownerVerified, studentVerified int
		if err := rows.Scan(&post.ID, &post.Author.ID, &post.Author.Name, &post.Author.Avatar,
			&ownerVerified, &studentVerified, &post.Content, &imagesJSON, &createdAt); err != nil {
			return domain.CommunityFeed{}, err
		}
		post.Author.OwnerVerified = ownerVerified == 1
		post.Author.StudentVerified = studentVerified == 1
		post.Images = decodeStrings(imagesJSON)
		post.CreatedAt = parseTimestamp(createdAt)
		post.Mine = post.Author.ID == viewerID
		items = append(items, post)
	}
	if err := rows.Err(); err != nil {
		return domain.CommunityFeed{}, err
	}
	feed := domain.CommunityFeed{Items: items}
	if len(items) > limit {
		feed.HasMore = true
		feed.Items = items[:limit]
		last := feed.Items[len(feed.Items)-1]
		feed.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(timestamp(last.CreatedAt) + "|" + last.ID))
	}
	return feed, nil
}

func (s *Store) CreateCommunityPost(authorID string, req domain.CreatePostRequest, now time.Time) (domain.CommunityPost, error) {
	user, err := s.userByID(authorID)
	if err != nil {
		return domain.CommunityPost{}, err
	}
	content := strings.TrimSpace(req.Content)
	images := nonEmptyStrings(req.Images)
	if content == "" && len(images) == 0 {
		return domain.CommunityPost{}, validation("动态内容和图片至少填写一项")
	}
	if len([]rune(content)) > 500 {
		return domain.CommunityPost{}, validation("动态内容不能超过 500 字")
	}
	if len(images) > 9 {
		return domain.CommunityPost{}, validation("每条动态最多 9 张图片")
	}
	for _, image := range images {
		if !strings.HasPrefix(image, "/uploads/") {
			return domain.CommunityPost{}, validation("图片必须先上传到平台")
		}
	}
	post := domain.CommunityPost{
		ID: newID("post"),
		Author: domain.CommunityAuthor{
			ID: user.ID, Name: user.Name, Avatar: user.Avatar,
			OwnerVerified: user.OwnerVerified, StudentVerified: user.StudentVerified,
		},
		Content: content, Images: images, CreatedAt: now, Mine: true,
	}
	if _, err := s.db.Exec(`
		INSERT INTO community_posts(id, author_id, content, images_json, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, post.ID, authorID, post.Content, marshalJSON(post.Images), timestamp(now)); err != nil {
		return domain.CommunityPost{}, err
	}
	return post, nil
}

func (s *Store) RecordUpload(ownerID, filename, originalName, mimeType string, size int64, now time.Time) error {
	if _, err := s.userByID(ownerID); err != nil {
		return err
	}
	_, err := s.db.Exec(`
		INSERT INTO uploads(id, owner_id, filename, original_name, mime_type, size_bytes, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
	`, newID("upload"), ownerID, filename, originalName, mimeType, size, timestamp(now))
	return err
}

func (s *Store) UploadOwner(filename string) (string, error) {
	var ownerID string
	err := s.db.QueryRow(`SELECT owner_id FROM uploads WHERE filename = ?`, filename).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", notFound("上传文件不存在")
	}
	return ownerID, err
}
