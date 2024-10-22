package utils

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sort"

	"github.com/stratosnet/sds/framework/crypto"
)

var rng *rand.Rand

func init() {
	var seed [8]byte
	_, err := cryptorand.Read(seed[:])
	if err != nil {
		panic("cannot set random seed from cryptographically secure random number")
	}
	seedInt := int64(binary.LittleEndian.Uint64(seed[:]))
	rng = rand.New(rand.NewSource(seedInt))
}

// GenerateRandomNumber generate (count) random numbers without repetition in the interval [start, end)
func GenerateRandomNumber(start int, end int, count int) []int {
	spread := end - start
	if end < start || spread < count || count == 0 {
		return nil
	}

	nums := make([]int, count)
	taken := make(map[int]bool)
	for i := 0; i < count; i++ {
		num := 0
		for {
			num = rng.Intn(spread) + start
			if !taken[num] {
				break
			}
		}
		taken[num] = true
		nums[i] = num
	}

	return nums
}

// GetRandomString between [0-9a-zA-Z]
func GetRandomString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}

	for i := 0; i < length; i++ {
		result = append(result, bytes[rng.Intn(len(bytes))])
	}
	return string(result)
}

type WeightedItem interface {
	Weight() float64
}

// WeightedRandomSelect selects a random item from the list (with probability proportional to item weight).
// If the seed is empty, it will use a cryptographically secure random seed
func WeightedRandomSelect(items []WeightedItem, seed string) (int, WeightedItem) {
	if len(items) == 0 {
		return 0, nil
	}

	rng := RandFromSeed(seed)
	weights := normalizeAndCumulateWeights(items)
	if len(weights) == 0 {
		// All nodes have 0 or negative weight
		selectedIndex := rng.Intn(len(items))
		return selectedIndex, items[selectedIndex]
	}

	randValue := rng.Float64()
	selectedIndex := sort.Search(len(weights), func(i int) bool { return randValue <= weights[i].weight })
	if selectedIndex == len(weights) {
		selectedIndex = rng.Intn(len(weights))
	}

	originalIndex := weights[selectedIndex].index
	return originalIndex, items[originalIndex]
}

// WeightedRandomSelectMultiple selects random items from the list (with probability proportional to item weight).
// If the seed is empty, it will use a cryptographically secure random seed
func WeightedRandomSelectMultiple(items []WeightedItem, count int, seed string) ([]int, []WeightedItem) {
	rng := RandFromSeed(seed)
	weights, sum := filterAndSum(items)
	if len(weights) == 0 {
		// All nodes have 0 or negative weight
		return nil, nil
	}
	if count > len(weights) {
		count = len(weights)
	}

	var selectedIndices []int
	var selectedItems []WeightedItem
	for i := 0; i < count; i++ {
		randValue := rng.Float64() * sum
		cumulated := float64(0)
		for j, w := range weights {
			cumulated += w.weight
			if cumulated >= randValue {
				originalIndex := weights[j].index
				selectedIndices = append(selectedIndices, originalIndex)
				selectedItems = append(selectedItems, items[originalIndex])

				if len(weights) > 1 {
					// Update weights for next selection
					sum -= w.weight
					if j < len(weights)-1 {
						weights[j] = weights[len(weights)-1]
					}
					weights = weights[:len(weights)-1]
				}
				break
			}
		}
	}
	return selectedIndices, selectedItems
}

type indexedWeight struct {
	index  int
	weight float64
}

func normalizeAndCumulateWeights(items []WeightedItem) []indexedWeight {
	weights, sum := filterAndSum(items)
	if sum == 0 {
		return weights
	}

	cumulated := float64(0)
	for i := range weights {
		w := weights[i].weight / sum
		weights[i].weight = w + cumulated
		cumulated += w
	}
	return weights
}

func filterAndSum(items []WeightedItem) ([]indexedWeight, float64) {
	var weights []indexedWeight
	sum := float64(0)
	for i, item := range items {
		weight := item.Weight()
		if weight <= 0 {
			continue // Exclude items with negative or zero weight as they can't be selected
		}
		sum += weight
		weights = append(weights, indexedWeight{
			index:  i,
			weight: weight,
		})
	}

	return weights, sum
}

func RandFromSeed(seed string) *rand.Rand {
	if seed == "" {
		return rng
	}
	seedInt := int64(crypto.CalcCRC32([]byte(crypto.CalcHash([]byte(seed)))))
	return rand.New(rand.NewSource(seedInt))
}
