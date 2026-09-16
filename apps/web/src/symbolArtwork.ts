import lingqianBanknote from './assets/symbols/lingqian-banknote.png'
import lingqianCashStack from './assets/symbols/lingqian-cash-stack.png'
import lingqianDiamond from './assets/symbols/lingqian-diamond.png'
import lingqianDogCoin from './assets/symbols/lingqian-dog-coin.png'

import streetLatiao from './assets/symbols/street/latiao.png'
import streetCola from './assets/symbols/street/cola.png'
import streetIcePop from './assets/symbols/street/ice-pop.png'
import streetIceCream from './assets/symbols/street/ice-cream.png'
import streetToyCar from './assets/symbols/street/toy-car.png'
import streetTv from './assets/symbols/street/tv.png'
import streetGame from './assets/symbols/street/game.png'

import arcadeMarble from './assets/symbols/arcade/marble.png'
import arcadeGlove from './assets/symbols/arcade/glove.png'
import arcadeRacer from './assets/symbols/arcade/racer.png'
import arcadePlane from './assets/symbols/arcade/plane.png'
import arcadeCrown from './assets/symbols/arcade/crown.png'

import mineCyan from './assets/symbols/mine/cyan.png'
import mineRed from './assets/symbols/mine/red.png'
import minePurple from './assets/symbols/mine/purple.png'
import mineGold from './assets/symbols/mine/gold.png'

import seaNet from './assets/symbols/sea/net.png'
import seaSeaweed from './assets/symbols/sea/seaweed.png'
import seaBoot from './assets/symbols/sea/boot.png'
import seaBottle from './assets/symbols/sea/bottle.png'
import seaAnchor from './assets/symbols/sea/anchor.png'
import seaPearl from './assets/symbols/sea/pearl.png'
import seaChest from './assets/symbols/sea/chest.png'
import seaCrown from './assets/symbols/sea/crown.png'

import diamondCracked from './assets/symbols/diamond/cracked.png'
import diamondIndustrial from './assets/symbols/diamond/industrial.png'
import diamondJewelry from './assets/symbols/diamond/jewelry.png'
import diamondRare from './assets/symbols/diamond/rare.png'
import diamondSelected from './assets/symbols/diamond/selected.png'
import diamondCollection from './assets/symbols/diamond/collection.png'
import diamondRoyal from './assets/symbols/diamond/royal.png'
import diamondEternal from './assets/symbols/diamond/eternal.png'

import allInSkull from './assets/symbols/all-in/skull.png'
import allInScythe from './assets/symbols/all-in/scythe.png'
import allInAngel from './assets/symbols/all-in/angel.png'

const artworkByTicket: Record<string, Record<string, string>> = {
  'lingqian-ticket': {
    '狗头金币': lingqianDogCoin,
    '钞票': lingqianBanknote,
    '碎钻石': lingqianDiamond,
    '钞票堆': lingqianCashStack,
  },
  'street-store': {
    '辣条': streetLatiao,
    '可乐': streetCola,
    '冰棍': streetIcePop,
    '雪糕': streetIceCream,
    '玩具车': streetToyCar,
    '小电视': streetTv,
    '游戏机': streetGame,
  },
  'arcade-challenge': {
    '弹珠': arcadeMarble,
    '拳套': arcadeGlove,
    '赛车': arcadeRacer,
    '飞机': arcadePlane,
    '街机皇冠': arcadeCrown,
  },
  'gold-mine': {
    '青晶簇': mineCyan,
    '红晶簇': mineRed,
    '紫晶簇': minePurple,
    '金色矿石': mineGold,
  },
  'deep-sea-salvage': {
    '空网': seaNet,
    '海草团': seaSeaweed,
    '破皮靴': seaBoot,
    '漂流瓶': seaBottle,
    '生锈船锚': seaAnchor,
    '珍珠贝': seaPearl,
    '黄金宝箱': seaChest,
    '海神王冠': seaCrown,
  },
  'eternal-color-diamond': {
    '裂纹级': diamondCracked,
    '工业级': diamondIndustrial,
    '珠宝级': diamondJewelry,
    '稀有级': diamondRare,
    '精选级': diamondSelected,
    '典藏级': diamondCollection,
    '皇室级': diamondRoyal,
    '永恒级': diamondEternal,
  },
  'all-in': {
    '骷髅头': allInSkull,
    '镰刀': allInScythe,
    '天使': allInAngel,
  },
}

export function symbolArtworkFor(cardCode: string, symbol: string) {
  return artworkByTicket[cardCode]?.[symbol.replace(/^目标·/, '')]
}
