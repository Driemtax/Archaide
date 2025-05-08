package character

type Character struct {
	Name     string
	ImageUrl string
}

var index int = 0

var characters []Character = []Character{
	{Name: "Adrian", ImageUrl: "https://api.dicebear.com/9.x/adventurer/svg?seed=Adrian&flip=true"},
	{Name: "Brian", ImageUrl: "https://api.dicebear.com/9.x/adventurer/svg?seed=Brian&flip=true"},
	{Name: "Amaya", ImageUrl: "https://api.dicebear.com/9.x/adventurer/svg?seed=Amaya&flip=true"},
	{Name: "Easton", ImageUrl: "https://api.dicebear.com/9.x/adventurer/svg?seed=Easton&flip=true"},
	{Name: "Chase", ImageUrl: "https://api.dicebear.com/9.x/adventurer/svg?seed=Chase&flip=true"},
	{Name: "Avery", ImageUrl: "https://api.dicebear.com/9.x/adventurer/svg?seed=Avery&flip=true"},
	{Name: "Alexander", ImageUrl: "https://api.dicebear.com/9.x/adventurer/svg?seed=Alexander&flip=true"},
}

func GetCharacter() *Character {
	newCharacter := &characters[index]
	index++
	if index >= len(characters) {
		index = 0
	}
	return newCharacter
}
