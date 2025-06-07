package abm

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"slices"
	"testing"
)

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestCountCommonElements(t *testing.T) {
	{
		v1 := []int{1, 2, 3, 5}
		v2 := []int{0, 2, 4}
		count := CountCommon(v1, v2)
		assert.Equal(t, count, 1, "countCommon returns 1 common element")
	}
	{
		v1 := []int{24, 25, 26, 27, 31, 32, 36, 40, 52, 58, 59, 60, 66, 68, 109}
		v2 := []int{24, 25, 26, 27, 31, 32, 36, 40, 52, 58, 59, 60, 66, 68, 109}
		count := CountCommon(v1, v2)
		assert.Equal(t, len(v1), 15, "v1 has correct number of elems")
		assert.Equal(t, len(v2), 15, "v2 has correct number of elems")
		assert.Equal(t, count, 15, "countCommon calculates equal slices")
	}
}

func TestSetAncestorsGeneral(t *testing.T) {
	const GENERATIONS = 4
	parameters := Parameters{
		SimulationId: 3,
		NumAgents:    20,
		Generations:  GENERATIONS,
		GrowthRate:   1.01,
		MatingK:      50,
		Compatible:   false,
	}
	simulation := NewSimulation(&parameters)
	simulation.Simulate()
	assert.Equal(t, len(simulation.agents) > 20, true, "At least 21 agents")
	generation := simulation.agents[len(simulation.agents)-1].generation
	counter := 0
	simulation.setAncestorsCurrGen()
	for _, agent := range simulation.agents {
		if agent.generation == generation {
			require.Equal(t, len(agent.ancestorSet) > 0, true, "ancestor set has elements for last generation agent")
			require.Equal(t, len(agent.ancestorVec) > 0, true, "ancestor vector has elements for last generation agent")
			require.Equal(t, len(agent.ancestorSet), len(agent.ancestorVec), "set and vector have same number of elemnts")
			counter++
		} else {
			require.Equal(t, len(agent.ancestorSet), 0, "ancestor set has 0 elements for not last generation agent")
			require.Equal(t, len(agent.ancestorVec), 0, "ancestor vector has 0 elements for not last generation agent")
		}
	}
	assert.Equal(t, counter > 0, true, "some agents exist")
}

func setupSim(t *testing.T) *Simulation {
	agents := []Agent{
		{
			id:         0,
			generation: 1,
			sex:        MALE,
			mother:     0,
			father:     0,
			children:   []int{2, 3, 4},
		},
		{
			id:         1,
			generation: 0,
			sex:        MALE,
			mother:     0,
			father:     0,
			children:   []int{2, 3, 4},
		},
		{
			id:         2,
			generation: 1,
			sex:        FEMALE,
			mother:     0,
			father:     1,
		},
		{
			id:         3,
			generation: 1,
			sex:        MALE,
			mother:     0,
			father:     1,
			children:   []int{5, 6, 7, 8},
		},
		{
			id:         4,
			generation: 1,
			sex:        MALE,
			mother:     0,
			father:     1,
			children:   []int{5, 6, 7, 8},
		},
		{
			id:         5,
			generation: 2,
			sex:        FEMALE,
			mother:     3,
			father:     4,
			children:   []int{9, 10},
		},
		{
			id:         6,
			generation: 2,
			sex:        MALE,
			mother:     3,
			father:     4,
			children:   []int{11, 12, 13},
		},
		{
			id:         7,
			generation: 2,
			sex:        FEMALE,
			mother:     3,
			father:     4,
			children:   []int{9, 10},
		},
		{
			id:         8,
			generation: 2,
			sex:        MALE,
			mother:     3,
			father:     4,
			children:   []int{11, 12, 13},
		},
		{
			id:         9,
			generation: 3,
			sex:        MALE,
			mother:     5,
			father:     7,
		},
		{
			id:         10,
			generation: 3,
			sex:        FEMALE,
			mother:     5,
			father:     7,
		},
		{
			id:         11,
			generation: 3,
			sex:        FEMALE,
			mother:     8,
			father:     6,
		},
		{
			id:         12,
			generation: 3,
			sex:        FEMALE,
			mother:     8,
			father:     6,
		},
		{
			id:         13,
			generation: 3,
			sex:        FEMALE,
			mother:     8,
			father:     6,
		},
	}
	parameters := NewParameters()
	simulation := NewSimulation(&parameters)
	simulation.agents = agents
	simulation.setCurrGen()
	assert.Equal(t, len(simulation.agents), 14, "Correct number of agents")
	assert.Equal(t, simulation.startCurrGen, 9, "Start current gen is correct")
	assert.Equal(t, len(simulation.currGen), 5, "Current gen has correct number of agents")
	return simulation
}

func setToVec(m map[int]struct{}) []int {
	result := make([]int, 0)
	for key := range m {
		result = append(result, key)
	}
	return result
}

func TestSetAncestorsSpecific(t *testing.T) {
	simulation := setupSim(t)
	simulation.setAncestorsCurrGen()
	{
		assert.Equal(t, simulation.agents[9].id, 9, "ID being set correctly")
		assert.Equal(t,
			[]int{0, 1, 3, 4, 5, 7},
			simulation.agents[9].ancestorVec,
			"Expected entries in ancestor vec")

		vecFromSet := setToVec(simulation.agents[9].ancestorSet)
		slices.Sort(vecFromSet)
		assert.Equal(t, simulation.agents[9].ancestorVec, vecFromSet, "Set and vec are equal")
	}
	{
		agent := simulation.agents[len(simulation.agents)-1]
		assert.Equal(t,
			[]int{0, 1, 3, 4, 6, 8},
			agent.ancestorVec,
			"Expected entries in ancestor vec")

		vecFromSet := setToVec(agent.ancestorSet)
		slices.Sort(vecFromSet)
		assert.Equal(t, agent.ancestorVec, vecFromSet, "Set and vec are equal")
	}
}
