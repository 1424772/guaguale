import lingqian from './assets/scratch-coatings/lingqian-ticket-coating-v1.webp'
import streetStore from './assets/scratch-coatings/street-store-coating-v1.webp'
import arcade from './assets/scratch-coatings/arcade-challenge-coating-v1.webp'
import goldMine from './assets/scratch-coatings/gold-mine-coating-v1.webp'
import rocket from './assets/scratch-coatings/rocket-launch-coating-v1.webp'
import deepSea from './assets/scratch-coatings/deep-sea-salvage-coating-v1.webp'
import diamond from './assets/scratch-coatings/eternal-color-diamond-coating-v1.webp'
import allIn from './assets/scratch-coatings/all-in-coating-v1.webp'

const artworkByCardCode: Record<string, string> = {
  'lingqian-ticket': lingqian,
  'street-store': streetStore,
  'arcade-challenge': arcade,
  'gold-mine': goldMine,
  'rocket-launch': rocket,
  'deep-sea-salvage': deepSea,
  'eternal-color-diamond': diamond,
  'all-in': allIn,
}

export function scratchCoatingArtworkFor(cardCode: string) {
  return artworkByCardCode[cardCode] ?? lingqian
}
