package main

import (
	"flag"
	"nathangeffen/abm"
)

// Process the command line arguments and return values set in
// parameters struct.
func processFlags() abm.Parameters {
	params := abm.NewParameters()
	var p abm.Parameters
	flag.IntVar(&p.SimulationId, "id", params.SimulationId, "Id of simulation")
	flag.IntVar(&p.NumAgents, "agents", params.NumAgents, "Number of agents")
	flag.IntVar(&p.Generations, "generations", params.Generations, "Number of generations to run for")
	flag.Float64Var(&p.GrowthRate, "growth", params.GrowthRate, "Growth rate of population")
	flag.BoolVar(&p.Monogamous, "monog", params.Monogamous, "Agents are monogamous")
	flag.IntVar(&p.MatingK, "matingk", params.MatingK, "Number of agents to search for compatible match")
	flag.BoolVar(&p.Compatible, "compatible", params.Compatible, "choose compatible agents when mating")
	flag.IntVar(&p.NumGenes, "genes", params.NumGenes, "Number of genes per agent in initial generation")
	flag.Float64Var(&p.MutationRate, "mutation", params.MutationRate, "Gene mutation rate")
	flag.StringVar(&p.Analysis, "analysis", params.Analysis,
		`N - Number of ancestors
C - Number of common ancestors
D - Generation differences
G - Gene analysis`)
	flag.Parse()
	return p
}

func main() {
	parameters := processFlags()
	simulation := abm.NewSimulation(&parameters)
	simulation.Simulate()
	simulation.Analysis()
}
