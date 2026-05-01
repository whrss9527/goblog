package mysql

import (
	"encoding/json"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm"

	"goblog/internal/pkg/model"
	"goblog/internal/repository"
	"goblog/pkg/utils"
)

func (r *GormRepository) GetPosts(params repository.PostParams) ([]*model.Post, int64, error) {
	var posts []*model.Post
	query := r.db.Table(model.Post{}.TableName()).Where("status = 1")

	if len(params.CategoryId) > 0 {
		query = query.Where("category_id = ?", params.CategoryId)
	}

	if len(params.TagId) > 0 {
		query = query.Where("JSON_CONTAINS(tag_ids, ?)", params.TagId)
	}

	if len(params.Keyword) > 0 {
		keywordQuery := r.db.Where("title LIKE ?", "%"+params.Keyword+"%").
			Or("description LIKE ?", "%"+params.Keyword+"%")

		if len(params.Ids["ids"]) > 0 {
			keywordQuery = keywordQuery.Or("id IN (?)", params.Ids["ids"])
		}

		if len(params.Ids["category_ids"]) > 0 {
			keywordQuery = keywordQuery.Or("category_id IN (?)", params.Ids["category_ids"])
		}

		if len(params.Ids["tag_ids"]) > 0 {
			for _, tagId := range params.Ids["tag_ids"] {
				keywordQuery = keywordQuery.Or("JSON_CONTAINS(tag_ids, ?)", tagId)
			}
		}

		query = query.Where(keywordQuery)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Order("is_top DESC, created_at DESC")

	if params.PerPage > 0 && params.Page > 0 {
		offset := (params.Page - 1) * params.PerPage
		query = query.Offset(offset).Limit(params.PerPage)
	}

	err := query.Find(&posts).Error
	if err != nil {
		return nil, 0, err
	}
	for _, post := range posts {
		var tagIds []int
		if err := json.Unmarshal([]byte(post.TagIdString), &tagIds); err == nil {
			post.TagIds = tagIds
		}
	}
	return posts, total, nil
}

// GetPost 根据 ID 获取文章
func (r *GormRepository) GetPost(id string) (model.Post, error) {
	var post model.Post
	err := r.db.Table(model.Post{}.TableName()).
		Select("id, title, created_at, updated_at, category_id, tag_ids, views, description, word_count, identity").
		Where("id = ?", id).
		First(&post).Error
	if err != nil {
		return post, err
	}
	var tagIds []int
	err = json.Unmarshal([]byte(post.TagIdString), &tagIds)
	if err != nil {
		return post, err
	}
	post.TagIds = tagIds
	var content model.PostContent
	err = r.db.Table(model.PostContent{}.TableName()).
		Select("content").
		Where("post_id =?", post.Id).
		First(&content).Error
	if err != nil {
		return post, err
	}
	post.Content = content.Content
	return post, err
}

// GetPostByIdentity 根据 Identity 获取文章
func (r *GormRepository) GetPostByIdentity(identity string) (model.Post, error) {
	var post model.Post
	err := r.db.Table(model.Post{}.TableName()).
		Select("id, title, created_at, updated_at, category_id, tag_ids, views, description, word_count, identity").
		Where("identity = ?", identity).
		First(&post).Error
	if err != nil {
		return post, err
	}
	var tagIds []int
	err = json.Unmarshal([]byte(post.TagIdString), &tagIds)
	if err != nil {
		return post, err
	}
	post.TagIds = tagIds

	var content model.PostContent
	err = r.db.Table(model.PostContent{}.TableName()).
		Select("content").
		Where("post_id =?", post.Id).
		First(&content).Error
	if err != nil {
		return post, err
	}
	post.Content = content.Content

	return post, err
}

// IncrView 增加文章浏览量
func (r *GormRepository) IncrView(id string) error {
	return r.db.Model(&model.Post{}).Table(model.Post{}.TableName()).Where("id = ?", id).Update("views", gorm.Expr("views + 1")).Error
}

// PostDelete 删除文章
func (r *GormRepository) PostDelete(post model.Post) (string, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		return "", tx.Error
	}

	if err := tx.Table(model.Post{}.TableName()).Where("id = ?", post.Id).Delete(&model.Post{}).Error; err != nil {
		tx.Rollback()
		return "", err
	}

	if err := tx.Table(model.PostContent{}.TableName()).Where("post_id = ?", post.Id).Delete(&model.PostContent{}).Error; err != nil {
		tx.Rollback()
		return "", err
	}

	return post.Id, tx.Commit().Error
}

// PostSave 保存文章
func (r *GormRepository) PostSave(post model.Post) (string, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		return "", tx.Error
	}

	var exists bool
	if err := tx.Table(model.Post{}.TableName()).Model(&model.Post{}).Where("id = ?", post.Id).Select("count(*) > 0").Scan(&exists).Error; err != nil {
		tx.Rollback()
		return "", err
	}

	if exists {
		count := utils.GetTotalWords(post.Content)
		tagIds, _ := json.Marshal(post.TagIds)
		if err := tx.Table(model.Post{}.TableName()).Model(&model.Post{}).Where("id = ?", post.Id).Updates(map[string]interface{}{
			"title":       post.Title,
			"description": post.Description,
			"category_id": post.CategoryId,
			"tag_ids":     tagIds,
			"word_count":  count,
			"identity":    post.Identity,
		}).Error; err != nil {
			tx.Rollback()
			return "", err
		}

		if err := tx.Table(model.PostContent{}.TableName()).Model(&model.PostContent{}).Where("post_id = ?", post.Id).Update("content", post.Content).Error; err != nil {
			tx.Rollback()
			return "", err
		}
	} else {
		post.CreatedAt = time.Now()
		post.UpdatedAt = time.Now()
		post.Id = strings.Replace(uuid.NewV4().String(), "-", "", -1)
		count := utils.GetTotalWords(post.Content)
		tagIds, _ := json.Marshal(post.TagIds)
		if err := tx.Table(model.Post{}.TableName()).Create(&model.Post{
			Id:          post.Id,
			Title:       post.Title,
			Status:      post.Status,
			CreatedAt:   post.CreatedAt,
			UpdatedAt:   post.UpdatedAt,
			CategoryId:  post.CategoryId,
			TagIdString: string(tagIds),
			Views:       post.Views,
			Description: post.Description,
			WordCount:   count,
			Identity:    post.Identity,
		}).Error; err != nil {
			tx.Rollback()
			return "", err
		}

		if err := tx.Table(model.PostContent{}.TableName()).Create(&model.PostContent{
			PostId:  post.Id,
			Content: post.Content,
		}).Error; err != nil {
			tx.Rollback()
			return "", err
		}
	}

	return post.Id, tx.Commit().Error
}

// GetPostCountByTagId 根据标签 ID 获取文章数量
func (r *GormRepository) GetPostCountByTagId(id string) (int, error) {
	var count int64
	err := r.db.Table(model.Post{}.TableName()).Model(&model.Post{}).Where("JSON_CONTAINS(tag_ids, ?)", id).Count(&count).Error
	return int(count), err
}

// GetPostIdsByContent 根据内容获取文章 ID 列表
func (r *GormRepository) GetPostIdsByContent(content string) ([]string, error) {
	var postIds []string
	err := r.db.Table(model.PostContent{}.TableName()).Where("content LIKE ?", "%"+content+"%").Pluck("post_id", &postIds).Error
	return postIds, err
}

func (r *GormRepository) PostExist(id string) (bool, error) {
	var count int64
	err := r.db.Table(model.Post{}.TableName()).Model(&model.Post{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *GormRepository) GetPostsArchive() ([]*model.Post, error) {
	var posts []*model.Post
	err := r.db.Table(model.Post{}.TableName()).
		Select("id, title, created_at, identity").
		Where("status = 1").
		Order("created_at DESC").
		Find(&posts).Error
	return posts, err
}

type postWithContent struct {
	model.Post
	Content string `gorm:"column:content"`
}

func (r *GormRepository) GetPostsWithContent() ([]*model.Post, error) {
	var rows []postWithContent
	err := r.db.Table(model.Post{}.TableName()+" AS p").
		Select("p.*, pc.content").
		Joins("LEFT JOIN "+model.PostContent{}.TableName()+" AS pc ON pc.post_id = p.id").
		Where("p.status = 1").
		Order("p.is_top DESC, p.created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	posts := make([]*model.Post, 0, len(rows))
	for _, row := range rows {
		post := row.Post
		post.Content = row.Content
		var tagIds []int
		if err := json.Unmarshal([]byte(post.TagIdString), &tagIds); err == nil {
			post.TagIds = tagIds
		}
		posts = append(posts, &post)
	}
	return posts, nil
}
