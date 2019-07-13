package main

import (
	"log"
)

// ChildrenCount keeps track of a single URL: the number of children that have been found
// so far (from GET-ing their parent) vs the number of children that have been received
// down the pipeline
// when these two numbers are equal we know that we are done for this parent
type childrenCount struct {
	numberOfFoundChildren    int
	numberOfReceivedChildren int
}

type State struct {
	childrenCountMap map[string]childrenCount
}

func NewStateTracker() *State {
	return &State{
		childrenCountMap: make(map[string]childrenCount)}
}


func (s *State) InitialiseRoot(rootName string) {
	s.NewChildFound(rootName)
}

func (s *State) NewChildFound(childName string) {
	s.childrenCountMap[childName] = childrenCount{numberOfFoundChildren: -1, numberOfReceivedChildren: 0}
}

func (s *State) ChildOfParent(parentName string) {
	// update the counts
	oldCounts := s.childrenCountMap[parentName]
	newCounts := childrenCount{
		numberOfFoundChildren:    oldCounts.numberOfFoundChildren,
		numberOfReceivedChildren: oldCounts.numberOfReceivedChildren + 1}
	s.childrenCountMap[parentName] = newCounts
}

func (s *State) ParentExplored(parentName string, numberOfChildren int) {
	// mark as explored and update total count
	oldChildrenCount := s.childrenCountMap[parentName]
	newChildrenCount := childrenCount{
		numberOfReceivedChildren: oldChildrenCount.numberOfReceivedChildren,
		numberOfFoundChildren:    numberOfChildren}

	s.childrenCountMap[parentName] = newChildrenCount
}

func (s *State) IsAllExplored() bool {
	// check if everything has been explored
	allExplored := true
	for _, value := range s.childrenCountMap {
		allExplored = allExplored &&
			value.numberOfReceivedChildren == value.numberOfFoundChildren
	}
	log.Printf("childrenCountMap %#v", s.childrenCountMap)
	return allExplored
}
