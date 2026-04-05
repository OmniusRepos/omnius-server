package database

import (
	"strings"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

func (d *DB) ListCuratedLists(includeInactive bool) ([]models.CuratedList, error) {
	query := d.Model(&models.CuratedList{})
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}
	var lists []models.CuratedList
	err := query.Order("display_order ASC, name ASC").Find(&lists).Error
	return lists, err
}

func (d *DB) GetCuratedListByID(id uint) (*models.CuratedList, error) {
	var l models.CuratedList
	if err := d.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (d *DB) GetCuratedList(idOrSlug string) (*models.CuratedList, error) {
	var l models.CuratedList
	if err := d.Where("id = ? OR slug = ?", idOrSlug, idOrSlug).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (d *DB) GetCuratedListMovies(list *models.CuratedList) ([]models.Movie, error) {
	var movieIDs []uint
	d.Model(&models.CuratedListMovie{}).
		Where("list_id = ?", list.ID).
		Order("display_order ASC").
		Pluck("movie_id", &movieIDs)

	if len(movieIDs) > 0 {
		var movies []models.Movie
		for _, id := range movieIDs {
			if m, err := d.GetMovie(id); err == nil {
				movies = append(movies, *m)
			}
		}
		return movies, nil
	}

	filter := MovieFilter{
		Limit:         list.LimitCount,
		SortBy:        list.SortBy,
		OrderBy:       list.OrderBy,
		MinimumRating: list.MinimumRating,
		MinimumYear:   list.MinimumYear,
		MaximumYear:   list.MaximumYear,
		Genre:         list.Genre,
	}
	movies, _, err := d.ListMovies(filter)
	return movies, err
}

func (d *DB) CreateCuratedList(list *models.CuratedList) error {
	if list.Slug == "" {
		list.Slug = strings.ToLower(strings.ReplaceAll(list.Name, " ", "-"))
	}
	return d.Create(list).Error
}

func (d *DB) UpdateCuratedList(list *models.CuratedList) error {
	return d.Save(list).Error
}

func (d *DB) DeleteCuratedList(id uint) error {
	return d.Delete(&models.CuratedList{}, id).Error
}

func (d *DB) AddMovieToCuratedList(listID, movieID uint, order int) error {
	clm := models.CuratedListMovie{ListID: listID, MovieID: movieID, DisplayOrder: order}
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "list_id"}, {Name: "movie_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"display_order"}),
	}).Create(&clm).Error
}

func (d *DB) RemoveMovieFromCuratedList(listID, movieID uint) error {
	return d.Where("list_id = ? AND movie_id = ?", listID, movieID).Delete(&models.CuratedListMovie{}).Error
}
