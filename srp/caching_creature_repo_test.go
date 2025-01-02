package srp

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCachingCreatureRepo_CreateCreature(t *testing.T) {
	type createCreatureCall struct {
		expectedName        string
		expectedDescription string
		result              Creature
		err                 error
	}
	testCases := []struct {
		name               string
		inputName          string
		inputDescription   string
		expectedCreateCall createCreatureCall
		expectResultCached bool
		expectedResult     Creature
		expectedErr        error
	}{
		{
			name:             "happy path",
			inputName:        "bob",
			inputDescription: "bob likes testing",
			expectedCreateCall: createCreatureCall{
				expectedName:        "bob",
				expectedDescription: "bob likes testing",
				result: Creature{
					ID:          1234,
					Name:        "bob",
					Description: "bob likes testing",
				},
			},
			expectResultCached: true,
			expectedResult: Creature{
				ID:          1234,
				Name:        "bob",
				Description: "bob likes testing",
			},
		},
		{
			name:             "error case",
			inputName:        "bob",
			inputDescription: "bob likes testing",
			expectedCreateCall: createCreatureCall{
				expectedName:        "bob",
				expectedDescription: "bob likes testing",
				err:                 errors.New("some DB error here"),
			},
			expectResultCached: false,
			expectedErr:        errors.New("some DB error here"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			testCacheDuration := time.Millisecond * 250

			rawRepo := NewMockRawCreatureRepo(t)
			rawRepo.EXPECT().CreateCreature(mock.Anything, tc.expectedCreateCall.expectedName, tc.expectedCreateCall.expectedDescription).Return(tc.expectedCreateCall.result, tc.expectedCreateCall.err).Once()

			testInstance := NewCachingCreatureRepo(rawRepo, testCacheDuration)

			res, err := testInstance.CreateCreature(ctx, tc.inputName, tc.inputDescription)
			assert.Equal(t, tc.expectedResult, res)
			assert.Equal(t, tc.expectedErr, err)

			if tc.expectResultCached {
				// easiest way to verify things were cached is to utilize the Get function without providing behavior for the underlying rawRepos get call
				fromLookup, err := testInstance.GetCreature(ctx, res.ID)
				assert.NoError(t, err)
				assert.True(t, fromLookup.ResultFound)
				assert.Equal(t, res, fromLookup.Creature)
			}
		})
	}
}

func TestCachingCreatureRepo_GetCreature(t *testing.T) {
	type getCreatureCall struct {
		inputID int64
		result  CreatureLookupResult
		err     error
	}
	testCases := []struct {
		name                    string
		inputID                 int64
		expectedGetCreatureCall getCreatureCall
		expectedResult          CreatureLookupResult
		expectResultCached      bool
		expectedErr             error
	}{
		{
			name:    "happy path, found",
			inputID: 123,
			expectedGetCreatureCall: getCreatureCall{
				inputID: 123,
				result: CreatureLookupResult{
					ResultFound: true,
					Creature: Creature{
						ID:          123,
						Name:        "bob",
						Description: "likes testing",
					},
				},
			},
			expectedResult: CreatureLookupResult{
				ResultFound: true,
				Creature: Creature{
					ID:          123,
					Name:        "bob",
					Description: "likes testing",
				},
			},
			expectResultCached: true,
		},
		{
			name:    "happy path, not found",
			inputID: 123,
			expectedGetCreatureCall: getCreatureCall{
				inputID: 123,
				result: CreatureLookupResult{
					ResultFound: false,
				},
			},
			expectedResult: CreatureLookupResult{
				ResultFound: false,
			},
			expectResultCached: true,
		},
		{
			name:    "error retrieving",
			inputID: 123,
			expectedGetCreatureCall: getCreatureCall{
				inputID: 123,
				err:     errors.New("boom goes the DB"),
			},
			expectedErr:        errors.New("boom goes the DB"),
			expectResultCached: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			testCacheDuration := time.Millisecond * 250

			rawRepo := NewMockRawCreatureRepo(t)
			rawRepo.EXPECT().GetCreature(mock.Anything, tc.expectedGetCreatureCall.inputID).Return(tc.expectedGetCreatureCall.result, tc.expectedGetCreatureCall.err).Once()

			testInstance := NewCachingCreatureRepo(rawRepo, testCacheDuration)

			result, err := testInstance.GetCreature(ctx, tc.inputID)
			assert.Equal(t, tc.expectedResult, result)
			assert.Equal(t, tc.expectedErr, err)

			if tc.expectResultCached {
				// first repeat the call without an expect on the raw repo to verify the result was cached
				result, err := testInstance.GetCreature(ctx, tc.inputID)
				assert.Equal(t, tc.expectedResult, result)
				assert.Equal(t, tc.expectedErr, err)

				time.Sleep(testCacheDuration)
				// now lets add the expected call to our mock and repeat to verify cache expiration works
				rawRepo.EXPECT().GetCreature(mock.Anything, tc.expectedGetCreatureCall.inputID).Return(tc.expectedGetCreatureCall.result, tc.expectedGetCreatureCall.err).Once()
				result, err = testInstance.GetCreature(ctx, tc.inputID)
				assert.Equal(t, tc.expectedResult, result)
				assert.Equal(t, tc.expectedErr, err)
			}
		})
	}
}

func TestCachingCreatureRepo_GetCreature_Concurrency(t *testing.T) {
	ctx := context.Background()

	inputID := int64(12345)
	expectedResult := CreatureLookupResult{
		ResultFound: true,
		Creature: Creature{
			ID:          12345,
			Name:        "bob",
			Description: "bob is popular and gets requested often",
		},
	}

	rawRepo := NewMockRawCreatureRepo(t)
	rawRepo.EXPECT().GetCreature(mock.Anything, inputID).Return(expectedResult, nil).Once()

	testInstance := NewCachingCreatureRepo(rawRepo, time.Hour)

	selectCount := 3000
	barrier := sync.WaitGroup{}
	barrier.Add(1)
	wg := sync.WaitGroup{}
	for i := 0; i < selectCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			barrier.Wait()
			result, err := testInstance.GetCreature(ctx, inputID)
			require.NoError(t, err)
			assert.Equal(t, expectedResult, result)
		}()
	}
	barrier.Done()
	wg.Wait()
}
