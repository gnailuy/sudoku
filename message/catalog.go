package message

import "fmt"

// ArgumentSpec declares one named argument in a catalog contract.
type ArgumentSpec struct {
	Name string
	Kind ArgumentKind
}

// Definition declares one catalog entry and its ordered argument contract.
type Definition struct {
	Arguments []ArgumentSpec
	Render    func([]Argument) (string, error)
}

// Catalog renders validated locale-neutral descriptors into one language.
type Catalog struct {
	definitions map[string]Definition
}

// NewCatalog creates a catalog and fails fast on malformed definitions.
func NewCatalog(definitions map[string]Definition) (*Catalog, error) {
	copy := make(map[string]Definition, len(definitions))
	for id, definition := range definitions {
		if !validID(id) {
			return nil, fmt.Errorf("invalid message id %q", id)
		}
		if definition.Render == nil {
			return nil, fmt.Errorf("message %q has no renderer", id)
		}
		copy[id] = definition
	}
	return &Catalog{definitions: copy}, nil
}

// Render validates arguments and returns a complete descriptor with fallback text.
func (c *Catalog) Render(id string, arguments ...Argument) (Descriptor, error) {
	definition, ok := c.definitions[id]
	if !ok {
		return Descriptor{}, fmt.Errorf("unknown message id %q", id)
	}
	if len(arguments) != len(definition.Arguments) {
		return Descriptor{}, fmt.Errorf("message %q expects %d arguments, got %d", id, len(definition.Arguments), len(arguments))
	}
	for i, spec := range definition.Arguments {
		if arguments[i].Name != spec.Name {
			return Descriptor{}, fmt.Errorf("message %q argument %d expects name %q, got %q", id, i, spec.Name, arguments[i].Name)
		}
		if arguments[i].Kind != spec.Kind {
			return Descriptor{}, fmt.Errorf("message %q argument %q expects %q, got %q", id, spec.Name, spec.Kind, arguments[i].Kind)
		}
	}
	text, err := definition.Render(arguments)
	if err != nil {
		return Descriptor{}, fmt.Errorf("render message %q: %w", id, err)
	}
	descriptor := Descriptor{ID: id, Arguments: append([]Argument{}, arguments...), Text: text}
	if err := descriptor.Validate(); err != nil {
		return Descriptor{}, err
	}
	return descriptor, nil
}
