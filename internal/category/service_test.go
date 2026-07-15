package category

import (
	"context"
	"testing"

	"github.com/omerkoc/cicekci/pkg/database"
	"github.com/omerkoc/cicekci/pkg/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	pool := database.NewTestDB(t)
	return NewService(NewStore(pool)), context.Background()
}

func TestService_Create(t *testing.T) {
	svc, ctx := newTestService(t)

	c, err := svc.Create(ctx, CreateInput{
		Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true, IsFeatured: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "Doğum Günü", c.Name)
	assert.Equal(t, "dogum-gunu", c.Slug)
	assert.Equal(t, AxisOccasion, c.Axis)
	assert.True(t, c.IsFeatured)
}

func TestService_Create_InvalidAxis(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "Test", Axis: "gecersiz"})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_EmptyName(t *testing.T) {
	svc, ctx := newTestService(t)

	_, err := svc.Create(ctx, CreateInput{Name: "  ", Axis: AxisType})

	require.ErrorIs(t, err, errorsx.ErrInvalidInput)
}

func TestService_Create_DuplicateSlugGetsSuffix(t *testing.T) {
	svc, ctx := newTestService(t)

	first, err := svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisType})
	require.NoError(t, err)
	second, err := svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisOccasion})
	require.NoError(t, err)

	assert.Equal(t, "buket", first.Slug)
	assert.Equal(t, "buket-2", second.Slug)
}

func TestService_ListPublic_HidesInactive(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{Name: "Anneler Günü", Axis: AxisOccasion, IsActive: false})
	require.NoError(t, err)

	list, err := svc.ListPublic(ctx, nil)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Doğum Günü", list[0].Name)
}

func TestService_ListPublic_FiltersByAxis(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	occasion := AxisOccasion
	list, err := svc.ListPublic(ctx, &occasion)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Doğum Günü", list[0].Name)
}

// Spec §4.1: is_active=false her şeyi ezer — pasif kategori featured olsa bile görünmez.
func TestService_ListFeatured_InactiveOverridesFeatured(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{
		Name: "Anneler Günü", Axis: AxisOccasion, IsActive: false, IsFeatured: true,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{
		Name: "Doğum Günü", Axis: AxisOccasion, IsActive: true, IsFeatured: true,
	})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{
		Name: "Taziye", Axis: AxisOccasion, IsActive: true, IsFeatured: false,
	})
	require.NoError(t, err)

	list, err := svc.ListFeatured(ctx)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Doğum Günü", list[0].Name)
}

func TestService_ListAdmin_ShowsInactive(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Aktif", Axis: AxisType, IsActive: true})
	require.NoError(t, err)
	_, err = svc.Create(ctx, CreateInput{Name: "Pasif", Axis: AxisType, IsActive: false})
	require.NoError(t, err)

	list, err := svc.ListAdmin(ctx)

	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestService_Update_PartialFields(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{
		Name: "Buket", Axis: AxisType, IsActive: true, IsFeatured: false,
	})
	require.NoError(t, err)

	featured := true
	updated, err := svc.Update(ctx, c.ID, UpdateInput{IsFeatured: &featured})

	require.NoError(t, err)
	assert.Equal(t, "Buket", updated.Name, "isim değişmemeli")
	assert.True(t, updated.IsFeatured)
	assert.True(t, updated.IsActive, "is_active değişmemeli")
}

// Spec §4.2: kategori slug'ı isim değişince güncellenmez — kategori URL'leri sabit kalır.
func TestService_Update_NameDoesNotChangeSlug(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{Name: "Buket", Axis: AxisType, IsActive: true})
	require.NoError(t, err)

	newName := "Gül Buketi"
	updated, err := svc.Update(ctx, c.ID, UpdateInput{Name: &newName})

	require.NoError(t, err)
	assert.Equal(t, "Gül Buketi", updated.Name)
	assert.Equal(t, "buket", updated.Slug, "slug sabit kalmalı")
}

func TestService_Update_NotFound(t *testing.T) {
	svc, ctx := newTestService(t)

	name := "Yok"
	_, err := svc.Update(ctx, 9999, UpdateInput{Name: &name})

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_Delete(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{Name: "Silinecek", Axis: AxisType})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, c.ID))

	_, err = svc.GetPublicBySlug(ctx, "silinecek")
	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_Delete_NotFound(t *testing.T) {
	svc, ctx := newTestService(t)

	err := svc.Delete(ctx, 9999)

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_GetPublicBySlug_HidesInactive(t *testing.T) {
	svc, ctx := newTestService(t)
	_, err := svc.Create(ctx, CreateInput{Name: "Pasif", Axis: AxisType, IsActive: false})
	require.NoError(t, err)

	_, err = svc.GetPublicBySlug(ctx, "pasif")

	require.ErrorIs(t, err, errorsx.ErrNotFound)
}

func TestService_ProductCount_Empty(t *testing.T) {
	svc, ctx := newTestService(t)
	c, err := svc.Create(ctx, CreateInput{Name: "Boş", Axis: AxisType})
	require.NoError(t, err)

	count, err := svc.ProductCount(ctx, c.ID)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
