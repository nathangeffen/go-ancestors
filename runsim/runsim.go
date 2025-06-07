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
	flag.IntVar(&p.SimulationId, "id", params.SimulationId, "id of simulation")
	flag.IntVar(&p.NumAgents, "agents", params.NumAgents, "number of agents")
	flag.IntVar(&p.Generations, "generations", params.Generations, "number of generations to run for")
	flag.Float64Var(&p.GrowthRate, "growth", params.GrowthRate, "growth rate of population")
	flag.IntVar(&p.MatingK, "k", params.MatingK, "Number of agents to search for compatible match")
	flag.BoolVar(&p.Compatible, "compatible", params.Compatible, "choose compatible agents when mating")
	flag.Parse()
	return p
}

func main() {
	parameters := processFlags()
	simulation := abm.NewSimulation(&parameters)
	simulation.Simulate()
	simulation.Analysis()
}
