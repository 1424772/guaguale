import lingqian from './assets/concepts/lingqian-ticket-play-v1.webp'
import streetStore from './assets/concepts/jiejiao-store-ticket-concept-v1.webp'
import arcade from './assets/concepts/jieji-challenge-ticket-concept-v1.webp'
import goldMine from './assets/concepts/golden-mine-ticket-concept-v1.webp'
import rocket from './assets/concepts/rocket-launch-ticket-concept-v1.webp'
import deepSea from './assets/concepts/deep-sea-salvage-ticket-concept-v1.webp'
import diamond from './assets/concepts/eternal-color-diamond-ticket-concept-v1.webp'
import allIn from './assets/concepts/all-in-ticket-concept-v2.webp'

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

export function ticketArtworkFor(cardCode: string) {
  return artworkByCardCode[cardCode] ?? lingqian
}

