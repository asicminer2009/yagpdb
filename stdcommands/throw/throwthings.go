package throw

import (
	"math/rand"
)

func randomThing() string {
	return throwThings[rand.Intn(len(throwThings))]
}

// If you want somthing added submit a pr
var throwThings = []string{
	"anime girls",
	"b1nzy",
	"bad jokes",
	"a boom box",
	"garliko~ (from GBDS)",
	"old cheese",
	"heroin",
	"sadness",
	"depression",
	"an evil villain with a plan to destroy earth",
	"a superhero on his way to stop an evil villain",
	"jonas747#0001",
	"hot firemen",
	"fat idiots",
	"a hairy potter",
	"hentai",
	"a soft knife",
	"a sharp pillow",
	"love",
	"hate",
	"a tomato",
	"a time machine that takes 1 year to travel 1 year into the future",
	"sadness disguised as hapiness",
	"debt",
	"all your imaginary friends",
	"homelessness",
	"stupid bot commands",
	"bad bots",
	"ol' musky",
	"wednesday, my dudes",
	"wednesday frog",
	"flat earthers",
	"round earthers",
	"prequel memes",
	"logan paul and his dead body",
	"selfbots",
	"very big fish",
	"ur mum",
	"the biggest fattest vape",
	"donald trump's wall",
	"smash mouth - all star",
	"an error",
	"death",
	"tide pods",
	"a happy life with a good career and a nice family",
	"a sad life with a deadend career and a horrible family",
	"divorce papers",
	"an engagement ring",
	"yourself",
	"nothing",
	"insults",
	"compliments",
	"life advice",
	"scams",
	"nothing",
	"crappy code",
	"2 million ping",
	"myself",
	"easter eggs",
	"explosives",
	"a black hole",
	"cheap servers",
	"babies",
	"oranges",
	"eggs",
	"cream puffs",
	"plasma grenades",
	"human body parts",
	"Whitney",
	"a wedding",
	"a car",
	"a chair",
}
