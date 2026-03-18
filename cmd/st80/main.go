package main

import (
	"fmt"
	"os"

	"github.com/wesen/st80/pkg/image"
	"github.com/wesen/st80/pkg/objectmemory"
)

func main() {
	imagePath := "data/VirtualImage"
	if len(os.Args) > 1 {
		imagePath = os.Args[1]
	}

	fmt.Printf("Loading image: %s\n", imagePath)
	om, err := image.LoadImage(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading image: %v\n", err)
		os.Exit(1)
	}

	om.Dump()

	// Verify well-known objects from the Blue Book (p.576).
	fmt.Println("\n--- Guaranteed Pointers ---")
	checkObject(om, "nil", objectmemory.NilPointer)
	checkObject(om, "false", objectmemory.FalsePointer)
	checkObject(om, "true", objectmemory.TruePointer)
	checkObject(om, "SchedulerAssociation", objectmemory.SchedulerAssociationPointer)
	checkObject(om, "String class", objectmemory.ClassStringPointer)
	checkObject(om, "Array class", objectmemory.ClassArrayPointer)
	checkObject(om, "MethodContext class", objectmemory.ClassMethodContextPointer)
	checkObject(om, "BlockContext class", objectmemory.ClassBlockContextPointer)
	checkObject(om, "Point class", objectmemory.ClassPointPointer)
	checkObject(om, "LargePositiveInteger class", objectmemory.ClassLargePositiveIntegerPointer)
	checkObject(om, "Message class", objectmemory.ClassMessagePointer)
	checkObject(om, "Character class", objectmemory.ClassCharacterPointer)
	checkObject(om, "SpecialSelectors", objectmemory.SpecialSelectorsPointer)
	checkObject(om, "CharacterTable", objectmemory.CharacterTablePointer)

	fmt.Println("\n--- Class objects (from class.oops) ---")
	checkObject(om, "SmallInteger", objectmemory.ClassSmallIntegerPointer)
	checkObject(om, "CompiledMethod", objectmemory.ClassCompiledMethodPointer)
	checkObject(om, "Association", objectmemory.ClassAssociationPointer)
	checkObject(om, "Object", objectmemory.ClassObjectPointer)
	checkObject(om, "Symbol", objectmemory.ClassSymbolPointer)

	// Check SmallInteger encoding
	fmt.Println("\n--- SmallInteger encoding ---")
	fmt.Printf("  ZeroPointer (oop=%d): value=%d\n",
		objectmemory.ZeroPointer, objectmemory.SmallIntegerValue(objectmemory.ZeroPointer))
	fmt.Printf("  OnePointer (oop=%d): value=%d\n",
		objectmemory.OnePointer, objectmemory.SmallIntegerValue(objectmemory.OnePointer))
	fmt.Printf("  TwoPointer (oop=%d): value=%d\n",
		objectmemory.TwoPointer, objectmemory.SmallIntegerValue(objectmemory.TwoPointer))
	fmt.Printf("  MinusOnePointer (oop=%d): value=%d\n",
		objectmemory.MinusOnePointer, objectmemory.SmallIntegerValue(objectmemory.MinusOnePointer))

	// Try to read the SchedulerAssociation (OOP 8) - should be an Association
	fmt.Println("\n--- SchedulerAssociation (OOP 8) ---")
	if om.ValidOop(objectmemory.SchedulerAssociationPointer) {
		assocClass := om.FetchClassOf(objectmemory.SchedulerAssociationPointer)
		key := om.FetchPointer(0, objectmemory.SchedulerAssociationPointer)
		value := om.FetchPointer(1, objectmemory.SchedulerAssociationPointer)
		fmt.Printf("  class=0x%04X (expect Association=0x%04X), key=0x%04X, value=0x%04X\n",
			assocClass, objectmemory.ClassAssociationPointer, key, value)
	}
}

func checkObject(om *objectmemory.ObjectMemory, name string, oop uint16) {
	if objectmemory.IsSmallInteger(oop) {
		fmt.Printf("  %s (oop %d): SmallInteger value=%d\n",
			name, oop, objectmemory.SmallIntegerValue(oop))
		return
	}
	if !om.ValidOop(oop) {
		fmt.Printf("  %s (oop %d/0x%04X): INVALID/FREE\n", name, oop, oop)
		return
	}
	classOop := om.FetchClassOf(oop)
	wordLen := om.FetchWordLengthOf(oop)
	hasPtr := om.HasPointerFields(oop)
	fmt.Printf("  %s (oop %d/0x%04X): class=0x%04X, words=%d, pointers=%v\n",
		name, oop, oop, classOop, wordLen, hasPtr)
}
