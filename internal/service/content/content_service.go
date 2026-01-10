// Package content 提供内容管理服务
package content

import (
	"context"
	"errors"

	"github.com/dumeirei/smart-locker-backend/internal/models"
	"github.com/dumeirei/smart-locker-backend/internal/repository"
)

// ContentService 内容服务
type ContentService struct {
	articleRepo *repository.ArticleRepository
}

// NewContentService 创建内容服务
func NewContentService(articleRepo *repository.ArticleRepository) *ContentService {
	return &ContentService{
		articleRepo: articleRepo,
	}
}

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	Category   string  `json:"category" binding:"required"`
	Title      string  `json:"title" binding:"required"`
	Content    string  `json:"content" binding:"required"`
	CoverImage *string `json:"cover_image"`
	Sort       int     `json:"sort"`
}

// CreateArticle 创建文章
func (s *ContentService) CreateArticle(ctx context.Context, req *CreateArticleRequest) (*models.Article, error) {
	article := &models.Article{
		Category:    req.Category,
		Title:       req.Title,
		Content:     req.Content,
		CoverImage:  req.CoverImage,
		Sort:        req.Sort,
		IsPublished: false,
	}

	if err := s.articleRepo.Create(ctx, article); err != nil {
		return nil, err
	}

	return article, nil
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	Category   *string `json:"category"`
	Title      *string `json:"title"`
	Content    *string `json:"content"`
	CoverImage *string `json:"cover_image"`
	Sort       *int    `json:"sort"`
}

// UpdateArticle 更新文章
func (s *ContentService) UpdateArticle(ctx context.Context, id int64, req *UpdateArticleRequest) (*models.Article, error) {
	article, err := s.articleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Category != nil {
		article.Category = *req.Category
	}
	if req.Title != nil {
		article.Title = *req.Title
	}
	if req.Content != nil {
		article.Content = *req.Content
	}
	if req.CoverImage != nil {
		article.CoverImage = req.CoverImage
	}
	if req.Sort != nil {
		article.Sort = *req.Sort
	}

	if err := s.articleRepo.Update(ctx, article); err != nil {
		return nil, err
	}

	return article, nil
}

// GetArticle 获取文章详情
func (s *ContentService) GetArticle(ctx context.Context, id int64) (*models.Article, error) {
	return s.articleRepo.GetByID(ctx, id)
}

// GetArticleWithViewCount 获取文章详情并增加浏览量
func (s *ContentService) GetArticleWithViewCount(ctx context.Context, id int64) (*models.Article, error) {
	article, err := s.articleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 增加浏览量
	_ = s.articleRepo.IncrementViewCount(ctx, id)

	return article, nil
}

// DeleteArticle 删除文章
func (s *ContentService) DeleteArticle(ctx context.Context, id int64) error {
	return s.articleRepo.Delete(ctx, id)
}

// ArticleListRequest 文章列表请求
type ArticleListRequest struct {
	Category    string `form:"category"`
	IsPublished *bool  `form:"is_published"`
	Keyword     string `form:"keyword"`
	Page        int    `form:"page,default=1"`
	PageSize    int    `form:"page_size,default=20"`
}

// ListArticles 获取文章列表
func (s *ContentService) ListArticles(ctx context.Context, req *ArticleListRequest) ([]*models.Article, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize

	filters := &repository.ArticleListFilters{
		Category:    req.Category,
		IsPublished: req.IsPublished,
		Keyword:     req.Keyword,
	}

	return s.articleRepo.List(ctx, offset, req.PageSize, filters)
}

// ListPublishedArticles 获取已发布的文章列表
func (s *ContentService) ListPublishedArticles(ctx context.Context, category string, page, pageSize int) ([]*models.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	return s.articleRepo.ListPublished(ctx, category, offset, pageSize)
}

// PublishArticle 发布文章
func (s *ContentService) PublishArticle(ctx context.Context, id int64) error {
	article, err := s.articleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if article.IsPublished {
		return errors.New("文章已发布")
	}

	return s.articleRepo.Publish(ctx, id)
}

// UnpublishArticle 取消发布文章
func (s *ContentService) UnpublishArticle(ctx context.Context, id int64) error {
	article, err := s.articleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !article.IsPublished {
		return errors.New("文章未发布")
	}

	return s.articleRepo.Unpublish(ctx, id)
}

// GetArticlesByCategory 按分类获取文章
func (s *ContentService) GetArticlesByCategory(ctx context.Context, category string, limit int) ([]*models.Article, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.articleRepo.GetByCategoryAndSort(ctx, category, limit)
}

// GetCategoryCounts 获取分类统计
func (s *ContentService) GetCategoryCounts(ctx context.Context) (map[string]int64, error) {
	return s.articleRepo.CountByCategory(ctx)
}
