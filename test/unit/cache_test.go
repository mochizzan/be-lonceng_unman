package unit_test

import (
	"context"
	"testing"
	"time"

	"be-lonceng_unman/internal/model"
	"be-lonceng_unman/internal/services"

	"github.com/stretchr/testify/assert"
)

func newTestCache() *services.CacheService {
	return services.NewCacheService(context.Background())
}

func TestCache_SetGet(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	data := &model.KRSResponse{
		Status:      "success",
		Mahasiswa:   model.Mahasiswa{NPM: "2211700006", Nama: "TEST"},
		TahunAjaran: "2025/2026",
		Semester:    "GENAP",
		MataKuliah:  []model.MataKuliah{{No: 1, Kode: "SI40306", Nama: "TEST", SKS: 6}},
		TotalSKS:    6,
	}

	err := cache.Set(ctx, "krs:2211700006", data, 1*time.Hour)
	assert.NoError(t, err)

	cached, err := cache.Get(ctx, "krs:2211700006")
	assert.NoError(t, err)
	assert.Equal(t, data.Mahasiswa.NPM, cached.Mahasiswa.NPM)
	assert.Equal(t, data.Mahasiswa.Nama, cached.Mahasiswa.Nama)
}

func TestCache_Miss(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	_, err := cache.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")
}

func TestCache_Expired(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	data := &model.KRSResponse{
		Status:    "success",
		Mahasiswa: model.Mahasiswa{NPM: "2211700006", Nama: "TEST"},
	}

	err := cache.Set(ctx, "krs:expired", data, 1*time.Millisecond)
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = cache.Get(ctx, "krs:expired")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache expired")
}

func TestCache_Delete(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	data := &model.KRSResponse{
		Status:    "success",
		Mahasiswa: model.Mahasiswa{NPM: "2211700006", Nama: "TEST"},
	}

	err := cache.Set(ctx, "krs:delete", data, 1*time.Hour)
	assert.NoError(t, err)

	err = cache.Delete(ctx, "krs:delete")
	assert.NoError(t, err)

	_, err = cache.Get(ctx, "krs:delete")
	assert.Error(t, err)
}

func TestCache_ZeroTTL(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	data := &model.KRSResponse{Status: "success"}
	err := cache.Set(ctx, "krs:zero", data, 0)
	assert.Error(t, err)
}

func TestCache_NegativeTTL(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	data := &model.KRSResponse{Status: "success"}
	err := cache.Set(ctx, "krs:neg", data, -1*time.Hour)
	assert.Error(t, err)
}

func TestCache_Overwrite(t *testing.T) {
	cache := newTestCache()
	ctx := context.Background()

	data1 := &model.KRSResponse{Status: "success", Mahasiswa: model.Mahasiswa{NPM: "111", Nama: "FIRST"}}
	data2 := &model.KRSResponse{Status: "success", Mahasiswa: model.Mahasiswa{NPM: "222", Nama: "SECOND"}}

	_ = cache.Set(ctx, "krs:overwrite", data1, 1*time.Hour)
	_ = cache.Set(ctx, "krs:overwrite", data2, 1*time.Hour)

	cached, err := cache.Get(ctx, "krs:overwrite")
	assert.NoError(t, err)
	assert.Equal(t, "SECOND", cached.Mahasiswa.Nama)
}
