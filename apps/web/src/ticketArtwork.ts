import lingqian from './assets/concepts/lingqian-ticket-play-v1.webp'
import streetStore from './assets/concepts/jiejiao-store-ticket-concept-v1.webp'
import arcade from './assets/concepts/jieji-challenge-ticket-concept-v1.webp'
import goldMine from './assets/concepts/golden-mine-ticket-concept-v1.webp'
import rocket from './assets/concepts/rocket-launch-ticket-concept-v1.webp'
import deepSea from './assets/concepts/deep-sea-salvage-ticket-concept-v1.webp'
import diamond from './assets/concepts/eternal-color-diamond-ticket-concept-v1.webp'
import allIn from './assets/concepts/all-in-ticket-concept-v2.webp'
import lingqianUnopened from './assets/unopened/lingqian-ticket-unopened-v1.webp'
import streetStoreUnopened from './assets/unopened/street-store-unopened-v1.webp'
import arcadeUnopened from './assets/unopened/arcade-challenge-unopened-v1.webp'
import goldMineUnopened from './assets/unopened/gold-mine-unopened-v1.webp'
import rocketUnopened from './assets/unopened/rocket-launch-unopened-v1.webp'
import deepSeaUnopened from './assets/unopened/deep-sea-salvage-unopened-v1.webp'
import diamondUnopened from './assets/unopened/eternal-color-diamond-unopened-v1.webp'
import allInUnopened from './assets/unopened/all-in-unopened-v1.webp'

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

const unopenedArtworkByCardCode: Record<string, string> = {
  'lingqian-ticket': lingqianUnopened,
  'street-store': streetStoreUnopened,
  'arcade-challenge': arcadeUnopened,
  'gold-mine': goldMineUnopened,
  'rocket-launch': rocketUnopened,
  'deep-sea-salvage': deepSeaUnopened,
  'eternal-color-diamond': diamondUnopened,
  'all-in': allInUnopened,
}

export function ticketArtworkFor(cardCode: string) {
  return artworkByCardCode[cardCode] ?? lingqian
}

export function unopenedTicketArtworkFor(cardCode: string) {
  return unopenedArtworkByCardCode[cardCode] ?? lingqianUnopened
}
