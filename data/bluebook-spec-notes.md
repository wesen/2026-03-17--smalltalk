# Blue Book Specification Notes

Extracted from "Smalltalk-80: The Language and its Implementation" (Goldberg & Robson)
for implementing the VM. These are my implementation notes summarizing the spec.

## Bytecode Table (Chapter 28, p.596)

| Range | Bits | Function |
|-------|------|----------|
| 0-15 | 0000iiii | Push Receiver Variable #iiii |
| 16-31 | 0001iiii | Push Temporary Location #iiii |
| 32-63 | 001iiiii | Push Literal Constant #iiiii |
| 64-95 | 010iiiii | Push Literal Variable #iiiii |
| 96-103 | 01100iii | Pop and Store Receiver Variable #iii |
| 104-111 | 01101iii | Pop and Store Temporary Location #iii |
| 112-119 | 01110iii | Push special (receiver, true, false, nil, -1, 0, 1, 2) |
| 120-123 | 011110ii | Return (receiver, true, false, nil) From Message |
| 124-125 | 0111110i | Return Stack Top From (Message, Block) |
| 126-127 | 0111111i | unused |
| 128 | 10000000 jjkkkkkk | Extended Push (j=type, k=index) |
| 129 | 10000001 jjkkkkkk | Extended Store |
| 130 | 10000010 jjkkkkkk | Pop and Store (extended) |
| 131 | 10000011 jjjkkkkk | Send Literal Selector #kkkkk With jjj Arguments |
| 132 | 10000100 jjjjjjjj kkkkkkkk | Send Literal Selector #kkkkkkkk With jjjjjjjj Args |
| 133 | 10000101 jjjkkkkk | Send Literal Selector #kkkkk To Superclass With jjj Args |
| 134 | 10000110 jjjjjjjj kkkkkkkk | Send To Superclass With jjjjjjjj Args |
| 135 | 10000111 | Pop Stack Top |
| 136 | 10001000 | Duplicate Stack Top |
| 137 | 10001001 | Push Active Context |
| 138-143 | | unused |
| 144-151 | 10010iii | Jump iii+1 (1 through 8) |
| 152-159 | 10011iii | Pop and Jump On False iii+1 (1 through 8) |
| 160-167 | 10100iii jjjjjjjj | Jump (iii-4)*256+jjjjjjjj |
| 168-171 | 101010ii jjjjjjjj | Pop and Jump On True ii*256+jjjjjjjj |
| 172-175 | 101011ii jjjjjjjj | Pop and Jump On False ii*256+jjjjjjjj |
| 176-191 | 1011iiii | Send Arithmetic Message #iiii |
| 192-207 | 1100iiii | Send Special Message #iiii |
| 208-223 | 1101iiii | Send Literal Selector #iiii With No Arguments |
| 224-239 | 1110iiii | Send Literal Selector #iiii With 1 Argument |
| 240-255 | 1111iiii | Send Literal Selector #iiii With 2 Arguments |

## Arithmetic Messages (bytecodes 176-191)

+, -, <, >, <=, >=, =, ~=, *, /, \\, @, bitShift:, //, bitAnd:, bitOr:

## Special Messages (bytecodes 192-207)

at:, at:put:, size, next, nextPut:, atEnd, ==, class, blockCopy:,
value, value:, do:, new, new:, x, y

## Guaranteed Pointers (Chapter 27, p.575-576)

### SmallIntegers
- MinusOnePointer = 65535 (0xFFFF)
- ZeroPointer = 1
- OnePointer = 3
- TwoPointer = 5

### initializeGuaranteedPointers
- NilPointer = 2
- FalsePointer = 4
- TruePointer = 6
- SchedulerAssociationPointer = 8
- ClassStringPointer = 14
- ClassArrayPointer = 16
- ClassMethodContextPointer = 22
- ClassBlockContextPointer = 24
- ClassPointPointer = 26
- ClassLargePositiveIntegerPointer = 28
- ClassMessagePointer = 32
- ClassCharacterPointer = 40
- DoesNotUnderstandSelector = 42
- CannotReturnSelector = 44
- SpecialSelectorsPointer = 48
- CharacterTablePointer = 50
- MustBeBooleanSelector = 52

## Context Field Indices (Chapter 27, p.581)

### MethodContext
- 0: sender
- 1: instruction pointer (SmallInteger, 1-based byte index)
- 2: stack pointer (SmallInteger, offset from TempFrameStart)
- 3: method (CompiledMethod pointer)
- 4: (unused)
- 5: receiver
- 6+: temporary frame (arguments, then temporaries, then stack)

### BlockContext
- 0: caller
- 1: instruction pointer
- 2: stack pointer
- 3: argument count (SmallInteger — distinguishes from MethodContext)
- 4: initial IP
- 5: home (MethodContext pointer)
- 6+: stack

### Distinguishing contexts
MethodContext stores a CompiledMethod (object pointer) in field 3.
BlockContext stores argument count (SmallInteger) in field 3.
isBlockContext: field 3 is a SmallInteger (bit 0 = 1).

## Class Field Indices (Chapter 27, p.587)

- 0: superclass
- 1: message dictionary
- 2: instance specification

### Instance Specification (SmallInteger, p.590-591)
- bit 0: isPointers (1 = pointer fields)
- bit 1: isWords (1 = word-sized fields)
- bit 2: isIndexable (1 = has indexable fields)
- bits 4-14: number of fixed fields

## Message Dictionary (p.587-588)
MethodDictionary is an IdentityDictionary:
- field 0: tally
- field 1: method array (Array of CompiledMethods)
- fields 2+: selectors (Symbols, with nils for empty slots)

Selector at index i corresponds to method at index i-SelectorStart in the method array.
Dictionary size is always a power of 2 + SelectorStart.

## CompiledMethod Format (Chapter 27, p.577-580)

### Structure
- field 0: header (SmallInteger)
- fields 1..literalCount: literal frame
- remaining: bytecodes (accessed as bytes)

### Header (SmallInteger in field 0)
Bit layout of the 15-bit value (remember bit 0 is always 1 for SmallInteger):
- bits 0-2: flag value
- bits 3-7: temporary count (or field index for flag=6)
- bit 8: large context flag
- bits 9-14: literal count

### Flag Values
- 0-4: no primitive, flag = number of arguments
- 5: primitive return of self (0 arguments)
- 6: primitive return of instance variable (0 arguments, temp count = field index)
- 7: header extension (next-to-last literal is another SmallInteger with arg count and primitive index)

### Header Extension (when flag = 7)
The next-to-last literal is a SmallInteger:
- bits 2-6: argument count
- bits 7-14: primitive index

### Method Class
Last literal of any method with super sends is an Association whose value is the class
in whose method dictionary the method was found.

## Interpreter Main Loop (Chapter 28, p.594)

```
interpret
  [true] whileTrue: [self cycle]
cycle
  self checkProcessSwitch.
  currentBytecode <- self fetchByte.
  self dispatchOnThisBytecode
```

## dispatchOnThisBytecode (p.595)
```
(currentBytecode between: 0 and: 119) ifTrue: [self stackBytecode].
(currentBytecode between: 120 and: 127) ifTrue: [self returnBytecode].
(currentBytecode between: 128 and: 130) ifTrue: [self stackBytecode].
(currentBytecode between: 131 and: 134) ifTrue: [self sendBytecode].
(currentBytecode between: 135 and: 137) ifTrue: [self stackBytecode].
(currentBytecode between: 144 and: 175) ifTrue: [self jumpBytecode].
(currentBytecode between: 176 and: 255) ifTrue: [self sendBytecode]
```

## Message Sending (Chapter 28, p.604-606)

### sendSelector:argumentCount:
1. Set messageSelector and argumentCount
2. Get receiver from stack (below arguments)
3. Get receiver's class
4. Call findNewMethodInClass: (with method cache)
5. Call executeNewMethod

### executeNewMethod
1. Call primitiveResponse
2. If primitive failed, call activateNewMethod

### activateNewMethod
1. Create new MethodContext (small or large based on largeContextFlag)
2. Store sender = activeContext
3. Store IP = initialInstructionPointerOfMethod
4. Store SP = temporaryCount
5. Store method = newMethod
6. Transfer receiver + arguments from old context
7. Make new context active

## Return (Chapter 28, p.608-610)

### returnBytecode
- 120: return receiver to sender
- 121: return true to sender
- 122: return false to sender
- 123: return nil to sender
- 124: return stack top to sender
- 125: return stack top to caller (block return)

### returnValue:to:
Check for nil sender/IP (cannotReturn error), then:
1. increaseReferencesTo: result
2. returnToActiveContext: target
3. push result
4. decreaseReferencesTo: result

## Primitive Table (Chapter 29, p.612-615)

| Range | Category |
|-------|----------|
| 1-18 | SmallInteger arithmetic |
| 21-37 | LargePositiveInteger arithmetic (optional) |
| 40-54 | Float arithmetic |
| 60-67 | Array/Stream subscripting |
| 68-79 | Storage management |
| 80-89 | Control (blockCopy, value, perform, semaphores) |
| 90-109 | Input/Output |
| 110-127 | System (==, class, quit, etc.) |

### Key Primitives
- 1: SmallInteger +
- 2: SmallInteger -
- 3-8: SmallInteger comparisons (<, >, <=, >=, =, ~=)
- 9: SmallInteger *
- 10: SmallInteger / (exact only)
- 11: SmallInteger \\ (mod, floor toward -inf)
- 12: SmallInteger // (div, floor toward -inf)
- 13: SmallInteger quo: (truncate toward 0)
- 14-16: SmallInteger bitAnd:, bitOr:, bitXor:
- 17: SmallInteger bitShift:
- 18: Number @ (makePoint)
- 60: at:
- 61: at:put:
- 62: size
- 63: String at: (returns Character)
- 64: String at:put:
- 70: basicNew / new
- 71: basicNew: / new:
- 75: hash / asOop
- 80: blockCopy:
- 81: value / value: / value:value:
- 85: Semaphore signal
- 86: Semaphore wait
- 87: Process resume
- 88: Process suspend
- 110: Character = / ==
- 111: class

## Object Table Entry Format (Chapter 30)

Each entry is 2 words (4 bytes):
- Word 0 bits 15-8: reference count (8 bits)
- Word 0 bit 7: odd length flag
- Word 0 bit 6: pointer fields flag
- Word 0 bit 5: free entry flag
- Word 0 bits 4-1: segment number (4 bits)
- Word 0 bit 0: unused
- Word 1: location within segment

Full heap address = segment * 65536 + location

## Object Body Format

- Word 0: size (total words including size and class)
- Word 1: class (OOP of the class)
- Words 2+: fields

## Image File Format (wolczko.com/st80)

- Bytes 0-3: object space size in 16-bit words (big-endian uint32)
- Bytes 4-7: object table size in 16-bit words (big-endian uint32)
- Bytes 8-511: zero padding
- Bytes 512+: object space data (big-endian 16-bit words)
- Gap/padding between OS and OT
- Last otSize*2 bytes: object table data (big-endian 16-bit words)

Total: 596,128 bytes = 512 header + 517,760 OS + 384 gap + 77,472 OT
- 258,880 OS words, 38,736 OT words (19,368 entries)
- 18,391 used objects, 977 free entries
