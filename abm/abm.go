// Package abm implements an agent based model to simulate population growth
// over generations in order to increase understanding of ancestry and inheritance.

package abm

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"math"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
)

// These can be set on the command line
type Parameters struct {
	SimulationId int
	NumAgents    int
	Generations  int
	GrowthRate   float64
	Monogamous   bool
	MatingK      int
	NumGenes     int
	MutationRate float64
	Compatible   bool
	Analysis     string
}

// Sets the default values for the parameters
func NewParameters() Parameters {
	return Parameters{
		SimulationId: 0,
		NumAgents:    100,
		Generations:  4,
		GrowthRate:   1.01,
		Monogamous:   true,
		MatingK:      50,
		NumGenes:     10,
		MutationRate: 0.0,
		Compatible:   true,
		Analysis:     "NCDG",
	}
}

type Sex int

const (
	MALE   Sex = 0
	FEMALE Sex = 1
)

// Data structure for each individual in the simulation.
// We keep both an array and set of ancestors because sometimes
// one is more efficient to use than the other.
// Genes are of the form [0-9]+\-[0-9]+`*
// The first integer is the agent id. The second is the number of the gene.
// Each backtick represents a mutation.
type Agent struct {
	id          int
	generation  int
	sex         Sex
	mother      int
	father      int
	children    []int
	ancestorVec []int
	ancestorSet map[int]struct{}
	genes       []string
}

// Checks if two agents share a mother or father in which case they are siblings.
func isSibling(a, b *Agent) bool {
	if a.generation == 0 {
		return false
	}
	return a.mother == b.mother || a.father == b.father
}

// Check if two agents share a grandparent in which case they are cousins.
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

// Finds all the ancestors for a given agent. id is the id of the agent for whom to calculate
func setAncestors(agents []Agent, id int) {
	ancestorSet := make(map[int]struct{})
	ancestorVec := make([]int, 0, agents[id].generation*2)
	ancestorVec = append(ancestorVec, id)
	generation := agents[id].generation
	sp := 0
	for sp < len(ancestorVec) {
		curr := ancestorVec[sp]
		currGen := agents[curr].generation
		if currGen < 1 { // The zero generation has no ancestry
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

// Generic function to count the number of common elements in two arrays
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

// Calculates the number of generations back you need to go to find
// a common ancestor between two agents. Maximum value is generation of
// first agent.
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

// Used to keep track of agents that are in mating pool.
type selectedAgent struct {
	id    int
	mated bool
}

// Used to keep track of agents that will reproduce.
type matingPair struct {
	male   int
	female int
}

// Data structure used by the simulation engine to manage
// state.
type Simulation struct {
	// Unique for each simulation
	id     int
	agents []Agent
	// Indicies of agents in current generation, which is usually
	// the last one but can be specified.
	currGen []selectedAgent
	// Keeps track of the indices that demarcate end of generations
	genBdrys []int
	// Agents that are paired to reproduce
	matingPairs []matingPair
	// User specified parameters
	params Parameters
}

// Creates a new simulation
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
		for i := range parameters.NumGenes {
			agent.genes = append(agent.genes, fmt.Sprintf("%d-%d", agent.id, i))
		}
		simulation.agents = append(simulation.agents, agent)
	}
	// Set current generation
	simulation.genBdrys = append(simulation.genBdrys, len(simulation.agents))
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

// Fills the generation vector with the IDs of a specified generation
func (s *Simulation) setGen(generation, start int) {
	s.currGen = s.currGen[:0]
	for _, agent := range s.agents[start:] {
		if agent.generation == generation {
			selected := selectedAgent{
				id:    agent.id,
				mated: false,
			}
			s.currGen = append(s.currGen, selected)
		}
	}
}

// Fills the current_generation vector with the IDs of the given generation
func (s *Simulation) setCurrGen(gen int) {
	s.currGen = s.currGen[:0]
	if gen >= len(s.genBdrys) {
		return
	}
	var start int
	if gen == 0 {
		start = 0
	} else {
		start = s.genBdrys[gen-1]
	}
	for _, agent := range s.agents[start:s.genBdrys[gen]] {
		s.currGen = append(s.currGen, selectedAgent{agent.id, false})
	}
}

// Sets the ancestors for every agent in the given generation
func (s *Simulation) setAncestorsGen(gen int) {
	for i := s.genBdrys[gen-1]; i < s.genBdrys[gen]; i++ {
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

func newChild(agents []Agent, father, mother, numGenes, generation int, mutationRate float64) []Agent {
	var sex Sex
	if rand.Float64() < 0.5 {
		sex = MALE
	} else {
		sex = FEMALE
	}
	agent := Agent{
		id:         len(agents),
		generation: generation,
		sex:        sex,
		father:     father,
		mother:     mother,
	}
	for i := range numGenes {
		if rand.Float64() < 0.5 {
			agent.genes = append(agent.genes, agents[father].genes[i])
		} else {
			agent.genes = append(agent.genes, agents[mother].genes[i])
		}
		if mutationRate > 0.0 && rand.Float64() < mutationRate {
			agent.genes[len(agent.genes)-1] += "`"
		}
	}
	agents = append(agents, agent)
	agents[father].children = append(agents[father].children, agent.id)
	agents[mother].children = append(agents[mother].children, agent.id)
	return agents
}

// Makes children agents from the mating_pairs vector
func (s *Simulation) makeChildrenMonogamous(generation int) {
	iterations := int(math.Ceil(s.params.GrowthRate * float64(len(s.currGen))))
	for range iterations {
		pair := s.matingPairs[rand.Intn(len(s.matingPairs))]
		s.agents = newChild(s.agents, pair.male, pair.female, s.params.NumGenes, generation, s.params.MutationRate)
	}
}

// Mating strategy in which any given agent mates with at most one other agent
func (s *Simulation) monogamousMating(generation int) {
	s.pairAgents()
	if len(s.matingPairs) > 0 {
		s.makeChildrenMonogamous(generation + 1)
	} else {
		fmt.Fprintln(os.Stderr, s.id, "Error: No mating pairs for generation",
			generation)
	}
}

// Mating strategy in which agents to mate are repeatedly selected to mate with anyone.
func (s *Simulation) nonMonogamousMating(generation int) {
	iterations := int(math.Ceil(s.params.GrowthRate * float64(len(s.currGen))))
	var males, females []int
	for _, selected := range s.currGen {
		index := selected.id
		switch s.agents[index].sex {
		case MALE:
			males = append(males, index)
		case FEMALE:
			females = append(females, index)
		}
	}

	if len(males) == 0 {
		fmt.Fprintln(os.Stderr, s.id, "Error: No males to make generation",
			generation)
		return
	}

	if len(females) == 0 {
		fmt.Fprintln(os.Stderr, s.id, "Error: No females to make generation",
			generation)
		return
	}

	for range iterations {
		i := males[rand.Intn(len(males))]
		j := females[rand.Intn(len(females))]
		s.agents = newChild(s.agents, i, j, s.params.NumGenes, generation, s.params.MutationRate)
	}
}

// Creates an array of integers in simulation.genBdrys where each integer is
// one past the simulation.agents index of the last agent with the generation
// matching the index of the array. This should generally only be needed for
// testing purposes because the genBdrys array is maintained by the simulation
// engine as it generates a new generation of agents.
func (s *Simulation) SetGenBdrys() {
	s.genBdrys = s.genBdrys[:0]
	if len(s.agents) == 0 {
		return
	}
	gen := s.agents[0].generation
	for i := range len(s.agents) {
		if gen != s.agents[i].generation {
			gen = s.agents[i].generation
			s.genBdrys = append(s.genBdrys, i)
		}
	}
	s.genBdrys = append(s.genBdrys, len(s.agents))
}

// This is the simulation engine function
func (s *Simulation) Simulate() {
	s.setCurrGen(0)
	for i := range s.params.Generations {
		if len(s.currGen) == 0 {
			fmt.Println("No survivors for generation", i, ".")
			break

		}
		if len(s.currGen) == 1 {
			fmt.Println("Only one survivor in generation", i, ".")
			break
		}
		rand.Shuffle(len(s.currGen), func(x, y int) {
			s.currGen[x], s.currGen[y] = s.currGen[y], s.currGen[x]
		})
		if s.params.Monogamous {
			s.monogamousMating(i)
		} else {
			s.nonMonogamousMating(i)
		}
		s.genBdrys = append(s.genBdrys, len(s.agents))
		s.setCurrGen(i + 1)
	}
}

// Reports statistics on number of ancestors agents in the last generation have
func (s *Simulation) reportNumAncestors() {
	generation := s.agents[len(s.agents)-1].generation
	count := 0
	total := 0
	min_ := math.MaxInt
	max_ := math.MinInt
	start := s.genBdrys[generation-1]
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
	generation := s.agents[len(s.agents)-1].generation
	start := s.genBdrys[generation-1]
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
	pop := len(s.agents) - start
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

// Reports statistics on gene distribution across a slice of agents
func analyzeGenes(agents []Agent) {
	geneTable := make(map[string]int)
	individualTable := make(map[int]int)
	for _, agent := range agents {
		for _, gene := range agent.genes {
			geneTable[gene]++
			components := strings.Split(gene, "-")
			individual, err1 := strconv.Atoi(components[0])
			if err1 != nil {
				fmt.Printf("Error converting gene components to int")
			} else {
				individualTable[individual]++
			}
		}
	}
	generation := agents[0].generation
	fmt.Printf("Number of different genes in generation %v: %v\n", generation, len(geneTable))
	maxGene, maxGeneCnt := "", 0
	for k, v := range geneTable {
		if v > maxGeneCnt {
			maxGene, maxGeneCnt = k, v
		}
	}
	fmt.Printf("Most common gene: %s: %d\n", maxGene, maxGeneCnt)
	maxIndividual, maxIndividualCnt := 0, 0
	for k, v := range individualTable {
		if v > maxIndividualCnt {
			maxIndividual, maxIndividualCnt = k, v
		}
	}
	fmt.Printf("Number original individuals contributing to gene pool %d\n", len(individualTable))
	fmt.Printf("Most common individual %d %d\n", maxIndividual, maxIndividualCnt)
	// fmt.Printf("Debug: %+v\n%+v\n", geneTable, individualTable)
}

// Reports gene statistics for a simulation
func (s *Simulation) reportGenes() {
	if len(s.agents) == 0 {
		return
	}
	start := 0
	generation := s.agents[0].generation
	for i, agent := range s.agents {
		if agent.generation != generation {
			analyzeGenes(s.agents[start:i])
			start = i
			generation = agent.generation
		}
	}
	analyzeGenes(s.agents[start:])
}

// Reports statistics on the outcome of a simulation
func (s *Simulation) Analysis() {
	fmt.Printf("For simulation %v:\n", s.id)
	fmt.Printf("Parameters: %+v\n", s.params)
	if len(s.agents) == 0 {
		fmt.Printf("No agents in simulation")
		return
	}
	generation := s.agents[len(s.agents)-1].generation
	if generation == 0 {
		fmt.Printf("Only zero generation exists")
		return
	}
	s.setAncestorsGen(generation)

	if strings.Contains(s.params.Analysis, "N") {
		s.reportNumAncestors()
	}

	if strings.Contains(s.params.Analysis, "C") {
		s.reportCommonAncestors()
	}

	if strings.Contains(s.params.Analysis, "D") {
		s.reportGenDiff()
	}

	if strings.Contains(s.params.Analysis, "G") {
		s.reportGenes()
	}
}
