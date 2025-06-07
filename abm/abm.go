package abm

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"math"
	"math/rand"
	"os"
	"slices"
)

type Parameters struct {
	SimulationId int
	NumAgents    int
	Generations  int
	GrowthRate   float64
	MatingK      int
	Compatible   bool
}

func NewParameters() Parameters {
	return Parameters{
		SimulationId: 0,
		NumAgents:    100,
		Generations:  4,
		GrowthRate:   1.01,
		MatingK:      50,
		Compatible:   true,
	}
}

type Sex int

const (
	MALE   Sex = 0
	FEMALE Sex = 1
)

type Agent struct {
	id          int
	generation  int
	sex         Sex
	mother      int
	father      int
	children    []int
	ancestorVec []int
	ancestorSet map[int]struct{}
}

func isSibling(a, b *Agent) bool {
	if a.generation == 0 {
		return false
	}
	return a.mother == b.mother || a.father == b.father
}

func isCousin(agents []Agent, a, b *Agent) bool {
	if a.generation < 2 || b.generation < 2 {
		return false
	}
	aMother := agents[a.mother]
	aFather := agents[a.father]
	bMother := agents[b.mother]
	bFather := agents[b.father]

	return isSibling(&aMother, &bMother) || isSibling(&aMother, &bFather) ||
		isSibling(&aFather, &bMother) || isSibling(&aFather, &bFather)
}

func setAncestors(agents []Agent, id int) {
	ancestorSet := make(map[int]struct{})
	ancestorVec := make([]int, 0, agents[id].generation*2)
	ancestorVec = append(ancestorVec, id)
	generation := agents[id].generation
	sp := 0
	for sp < len(ancestorVec) {
		curr := ancestorVec[sp]
		currGen := agents[curr].generation
		if currGen < 1 {
			break
		}
		mother := agents[curr].mother
		father := agents[curr].father
		parents := [...]int{mother, father}
		for _, parent := range parents {
			if _, found := ancestorSet[parent]; !found {
				ancestorVec = append(ancestorVec, parent)
				ancestorSet[parent] = struct{}{}
			}
		}
		sp += 1
		if currGen < generation {
			generation = currGen
		}
	}
	slices.Sort(ancestorVec)
	ancestorVec = ancestorVec[:len(ancestorVec)-1] // Remove self
	agents[id].ancestorVec = ancestorVec
	agents[id].ancestorSet = ancestorSet
}

func CountCommon[S ~[]E, E constraints.Ordered](vecA S, vecB S) int {
	i := 0
	j := 0
	total := 0
	for i < len(vecA) && j < len(vecB) {
		if vecA[i] < vecB[j] {
			for i < len(vecA) && vecA[i] <= vecB[j] {
				if vecA[i] == vecB[j] {
					total++
				}
				i++
			}
		} else {
			for j < len(vecB) && vecB[j] <= vecA[i] {
				if vecB[j] == vecA[i] {
					total++
				}
				j++
			}
		}
	}
	return total
}

func generationDiff(agents []Agent, a *Agent, b *Agent) int {
	generationFound := 0
	for i := len(a.ancestorVec) - 1; i >= 0; i-- {
		index := a.ancestorVec[i]
		if _, found := b.ancestorSet[index]; found {
			generationFound = agents[index].generation
			break
		}
	}
	return a.generation - generationFound
}

type selectedAgent struct {
	id    int
	mated bool
}

type matingPair struct {
	male   int
	female int
}

type Simulation struct {
	id           int
	agents       []Agent
	currGen      []selectedAgent
	startCurrGen int
	matingPairs  []matingPair
	params       Parameters
}

func NewSimulation(parameters *Parameters) *Simulation {
	var simulation Simulation
	simulation.params = *parameters
	simulation.id = parameters.SimulationId
	// Create agents
	for i := range parameters.NumAgents {
		var sex Sex
		if rand.Float64() < 0.5 {
			sex = MALE
		} else {
			sex = FEMALE
		}
		agent := Agent{
			id:         i,
			generation: 0,
			sex:        sex,
			mother:     0,
			father:     0,
		}
		simulation.agents = append(simulation.agents, agent)
	}
	// Set current generation
	for i := range len(simulation.agents) {
		selectedAgent := selectedAgent{
			id:    i,
			mated: false,
		}
		simulation.currGen = append(simulation.currGen, selectedAgent)
	}
	return &simulation
}

// Checks if two agents are compatible for mating
func (s *Simulation) compatible(a, b *Agent) bool {
	if a.sex == b.sex || isSibling(a, b) || isCousin(s.agents, a, b) {
		return false
	}
	return true
}

// Fills the current_generation vector with the IDs of the latest generation
func (s *Simulation) setCurrGen() {
	s.currGen = s.currGen[:0]
	generation := s.agents[len(s.agents)-1].generation
	for _, agent := range s.agents[s.startCurrGen:] {
		if agent.generation == generation {
			selected := selectedAgent{
				id:    agent.id,
				mated: false,
			}
			s.currGen = append(s.currGen, selected)
		}
	}
	s.startCurrGen = s.currGen[0].id
}

func (s *Simulation) setAncestorsCurrGen() {
	for i := s.startCurrGen; i < len(s.agents); i++ {
		setAncestors(s.agents, i)
	}
}

// Helper function for pairAgents that makes a single pair
func makePair(agentA *Agent, agentB *Agent) matingPair {

	var pair matingPair
	if agentA.sex == MALE {
		pair.male = agentA.id
		pair.female = agentB.id
	} else {
		pair.male = agentB.id
		pair.female = agentA.id
	}
	return pair
}

// Creates pairs of compatible agents that will be used to generate children
func (s *Simulation) pairAgents() {
	s.matingPairs = s.matingPairs[:0]
	for i := range len(s.currGen) {
		agentA := &s.agents[s.currGen[i].id]
		if s.currGen[i].mated == true {
			continue
		}
		hi := min(len(s.currGen), i+s.params.MatingK)
		for j := i + 1; j < hi; j++ {
			if s.currGen[j].mated == true {
				continue
			}
			agentB := &s.agents[s.currGen[j].id]
			if s.params.Compatible == false || s.compatible(agentA, agentB) == true {
				pair := makePair(agentA, agentB)
				s.matingPairs = append(s.matingPairs, pair)
				s.currGen[i].mated = true
				s.currGen[j].mated = true
				break
			}
		}
	}
}

// Makes children agents from the mating_pairs vector
func (s *Simulation) makeChildren(generation int) {
	iterations := int(math.Ceil(s.params.GrowthRate * float64(len(s.currGen))))
	for range iterations {
		pair := s.matingPairs[rand.Intn(len(s.matingPairs))]
		var sex Sex
		if rand.Float64() > 0.5 {
			sex = MALE
		} else {
			sex = FEMALE
		}
		agent := Agent{
			id:         len(s.agents),
			generation: generation,
			sex:        sex,
			father:     s.agents[pair.male].id,
			mother:     s.agents[pair.female].id,
		}
		s.agents[pair.male].children = append(s.agents[pair.male].children, agent.id)
		s.agents[pair.female].children = append(s.agents[pair.male].children, agent.id)
		s.agents = append(s.agents, agent)
	}
}

// / This is the simulation engine function
func (s *Simulation) Simulate() {
	for i := range s.params.Generations {
		s.setCurrGen()
		if len(s.currGen) > 0 {
			rand.Shuffle(len(s.currGen), func(i, j int) {
				s.currGen[i], s.currGen[j] = s.currGen[j], s.currGen[i]
			})
			s.pairAgents()
			if len(s.matingPairs) > 0 {
				s.makeChildren(i + 1)
			} else {
				fmt.Println("No mating pairs for generation", i, ".")
				break
			}
		} else {
			fmt.Println("No survivors for generation", i, ".")
			break
		}
	}
	if len(s.agents) > 0 {
		s.setCurrGen()
	}
}

// / Reports statistics on number of ancestors agents in the last generation have
func (s *Simulation) reportNumAncestors() {
	generation := s.agents[len(s.agents)-1].generation
	count := 0
	total := 0
	min_ := math.MaxInt
	max_ := math.MinInt
	start := s.startCurrGen
	for _, agent := range s.agents[start:] {
		numAncestors := len(agent.ancestorVec)
		total += numAncestors
		count++
		if numAncestors < min_ {
			min_ = numAncestors
		}
		if numAncestors > max_ {
			max_ = numAncestors
		}
	}
	avg := math.Round(float64(total) / float64(count))
	fmt.Println("Number agents", len(s.agents))
	fmt.Println("Number agents  last generation ", count)
	fmt.Printf("Generations: %v Max possible ancestors %v\n", generation, math.Pow(2, float64(generation+1))-2)
	fmt.Printf("Min, max, mean number of ancestors for agents in last generation: %v %v %v\n", min_, max_, avg)
}

// Reports statistics on the number of common ancestors that agents in the last generation have
func (s *Simulation) reportCommonAncestors() {
	start := s.startCurrGen
	total := 0
	min_ := math.MaxInt
	max_ := math.MinInt
	for _, agent := range s.agents[start : len(s.agents)-1] {
		for j := agent.id + 1; j < len(s.agents); j++ {
			common := CountCommon(agent.ancestorVec, s.agents[j].ancestorVec)
			if common < min_ {
				min_ = common
			}
			if common > max_ {
				max_ = common
			}
			total += common
		}
	}
	pop := len(s.agents) - s.startCurrGen
	avg := math.Round(float64(total) / (float64(pop) * float64(pop) / 2.0))
	fmt.Printf("Min, max, mean number of common ancestors (for last generation): %v %v %v\n", min_, max_, avg)
}

// Reports statistics on the number of generations back you have to search to
// / find common ancestors of the agents in the last generation
func (s *Simulation) reportGenDiff() {
	lastGen := s.agents[len(s.agents)-1].generation
	if lastGen == 0 {
		fmt.Fprintf(os.Stderr, "There is only one generation.\n")
		return
	}
	count := 0
	total := 0
	min_ := math.MaxInt
	max_ := 0
	for i := len(s.agents) - 1; i >= 0; i-- {
		a := &s.agents[i]
		if a.generation != lastGen {
			break
		}
		count++
		for j := a.id - 1; j > 0; j-- {
			b := &s.agents[j]
			if b.generation != lastGen {
				break
			}
			difference := generationDiff(s.agents, a, b)
			if difference < min_ {
				min_ = difference
			}
			if difference > max_ {
				max_ = difference
			}
			total += difference
		}
	}
	avg := math.Round(float64(total) / (float64(count*count) / 2.0))
	fmt.Printf("Min, max, mean generation difference (for last generation): %v %v %v\n", min_, max_, avg)
}

// Reports statistics on the outcome of a simulation
func (s *Simulation) Analysis() {
	fmt.Printf("For simulation %v:\n", s.id)
	fmt.Printf("Parameters: %+v\n", s.params)
	s.setAncestorsCurrGen()
	s.reportNumAncestors()
	s.reportCommonAncestors()
	s.reportGenDiff()
}
