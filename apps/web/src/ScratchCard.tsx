import { useEffect, useRef, useState } from 'react'
import { symbolArtworkFor } from './symbolArtwork'

type ScratchCardProps = {
  cardCode: string
  cardName: string
  symbols: string[]
  scratchLevel: number
  prizeTier?: string
  onComplete: () => void
  onProgress?: (progress: number) => void
}

type CellBounds = { x: number; y: number; width: number; height: number }

const scratchEffects = [0, 6, 8, 11, 15, 21, 29, 40, 55, 75, 100]

const symbolVisuals: Record<string, { emoji: string; label: string }> = {
  '狗头金币': { emoji: '🐶', label: '狗头金币' },
  '钞票': { emoji: '💵', label: '钞票' },
  '碎钻石': { emoji: '💎', label: '碎钻石' },
  '钞票堆': { emoji: '💰', label: '钞票堆' },
  '辣条': { emoji: '🌶️', label: '辣条' },
  '可乐': { emoji: '🥤', label: '可乐' },
  '冰棍': { emoji: '🧊', label: '冰棍' },
  '雪糕': { emoji: '🍦', label: '雪糕' },
  '玩具车': { emoji: '🚗', label: '玩具车' },
  '小电视': { emoji: '📺', label: '小电视' },
  '游戏机': { emoji: '🎮', label: '游戏机' },
  '弹珠': { emoji: '🔵', label: '弹珠' },
  '拳套': { emoji: '🥊', label: '拳套' },
  '赛车': { emoji: '🏎️', label: '赛车' },
  '飞机': { emoji: '✈️', label: '飞机' },
  '街机皇冠': { emoji: '👑', label: '街机皇冠' },
  '青晶簇': { emoji: '🔷', label: '青晶簇' },
  '红晶簇': { emoji: '🔶', label: '红晶簇' },
  '紫晶簇': { emoji: '💜', label: '紫晶簇' },
  '金色矿石': { emoji: '🪨', label: '金色矿石' },
  '海神王冠': { emoji: '👑', label: '海神王冠' },
  '黄金宝箱': { emoji: '🧰', label: '黄金宝箱' },
  '珍珠贝': { emoji: '🦪', label: '珍珠贝' },
  '生锈船锚': { emoji: '⚓', label: '生锈船锚' },
  '漂流瓶': { emoji: '🍾', label: '漂流瓶' },
  '破皮靴': { emoji: '🥾', label: '破皮靴' },
  '海草团': { emoji: '🌿', label: '海草团' },
  '空网': { emoji: '🕸️', label: '空网' },
  '裂纹级': { emoji: '♢', label: '裂纹级' },
  '工业级': { emoji: '◆', label: '工业级' },
  '珠宝级': { emoji: '◈', label: '珠宝级' },
  '稀有级': { emoji: '◎', label: '稀有级' },
  '精选级': { emoji: '✦', label: '精选级' },
  '典藏级': { emoji: '♜', label: '典藏级' },
  '皇室级': { emoji: '♛', label: '皇室级' },
  '永恒级': { emoji: '∞', label: '永恒级' },
  '骷髅头': { emoji: '☠️', label: '骷髅头' },
  '镰刀': { emoji: '⚔️', label: '镰刀' },
  '天使': { emoji: '🪽', label: '天使' },
}

const diamondAppraisals = ['重量', '切工', '净度', '火彩', '稀有度']

export function getSymbolVisual(symbol: string) {
  const baseSymbol = symbol.replace(/^目标·/, '')
  const fuelValue = symbol.match(/^燃料 (\d)$/)?.[1]
  return fuelValue
    ? { emoji: `⛽${fuelValue}`, label: symbol }
    : symbolVisuals[baseSymbol] ?? { emoji: '✦', label: symbol }
}

export function TicketSymbolGlyph({ cardCode, symbol }: { cardCode: string; symbol: string }) {
  const baseSymbol = symbol.replace(/^目标·/, '')
  const artwork = symbolArtworkFor(cardCode, baseSymbol)
  if (artwork) {
    return <span className={`ticket-symbol-glyph ticket-symbol-glyph-artwork art-card-${cardCode}`} data-symbol={baseSymbol} aria-hidden="true"><img src={artwork} alt="" draggable={false} loading="eager" decoding="sync" /></span>
  }
  const fuelValue = symbol.match(/^燃料 (\d)$/)?.[1]
  if (fuelValue) return <span className={`ticket-symbol-glyph ticket-fuel-glyph fuel-${fuelValue}`} aria-hidden="true"><b>⚡</b><em>{fuelValue}</em></span>
  return <span className="ticket-symbol-glyph" aria-hidden="true">{getSymbolVisual(symbol).emoji}</span>
}

export function symbolClassName(symbol: string) {
  const names: Record<string, string> = {
    '裂纹级': 'cracked', '工业级': 'industrial', '珠宝级': 'jewelry', '稀有级': 'rare',
    '精选级': 'selected', '典藏级': 'collection', '皇室级': 'royal', '永恒级': 'eternal',
    '骷髅头': 'skull', '镰刀': 'scythe', '天使': 'angel',
  }
  return names[symbol] ?? 'standard'
}

const seaValueOrder = ['空网', '海草团', '破皮靴', '漂流瓶', '生锈船锚', '珍珠贝', '黄金宝箱', '海神王冠']
const diamondValueOrder = ['裂纹级', '工业级', '珠宝级', '稀有级', '精选级', '典藏级', '皇室级', '永恒级']

function matchingIndexes(symbols: string[], minimum: number) {
  const counts = new Map<string, number>()
  symbols.forEach((symbol) => counts.set(symbol, (counts.get(symbol) ?? 0) + 1))
  return new Set(symbols.flatMap((symbol, index) => (counts.get(symbol) ?? 0) >= minimum ? [index] : []))
}

export function symbolStateClassNames(cardCode: string, symbols: string[], index: number, complete = true) {
  if (!complete) return ''
  const classes: string[] = []
  if (cardCode === 'lingqian-ticket' && matchingIndexes(symbols, 2).has(index)) classes.push('winning-symbol')
  if (cardCode === 'street-store' && matchingIndexes(symbols, 3).has(index)) classes.push('winning-symbol')
  if (cardCode === 'arcade-challenge') {
    const rowStart = Math.floor(index / 3) * 3
    if (symbols[rowStart] === symbols[rowStart + 1] && symbols[rowStart] === symbols[rowStart + 2]) classes.push('winning-row')
  }
  if (cardCode === 'gold-mine') {
    const target = symbols[0]?.replace(/^目标·/, '')
    if (index === 0) classes.push('target-symbol')
    else if (target && symbols[index] === target) classes.push('target-match')
  }
  if (cardCode === 'deep-sea-salvage') {
    const highest = Math.max(...symbols.map((symbol) => seaValueOrder.indexOf(symbol)))
    if (seaValueOrder.indexOf(symbols[index]) === highest) classes.push('highest-value')
  }
  if (cardCode === 'eternal-color-diamond') {
    const lowest = Math.min(...symbols.map((symbol) => diamondValueOrder.indexOf(symbol)))
    if (diamondValueOrder.indexOf(symbols[index]) === lowest) classes.push('lowest-grade')
  }
  return classes.join(' ')
}

function getCellBounds(cardCode: string, count: number, columns: number, width: number, height: number): CellBounds[] {
  if (cardCode === 'street-store') {
    const gap = width * .014
    const cellWidth = width * .225
    const cellHeight = height * .455
    return Array.from({ length: count }, (_, index) => index < 6
      ? {
          x: width * .01 + (index % 3) * (cellWidth + gap),
          y: height * .02 + Math.floor(index / 3) * (cellHeight + height * .04),
          width: cellWidth,
          height: cellHeight,
        }
      : { x: width * .735, y: height * .15, width: width * .255, height: height * .7 })
  }
  if (cardCode === 'gold-mine') {
    return Array.from({ length: count }, (_, index) => index === 0
      ? { x: width * .01, y: height * .08, width: width * .32, height: height * .84 }
      : {
          x: width * .48 + ((index - 1) % 3) * width * .175,
          y: height * .05 + Math.floor((index - 1) / 3) * height * .49,
          width: width * .15,
          height: height * .42,
        })
  }
  if (cardCode === 'eternal-color-diamond') {
    const gap = 12
    const margin = 18
    const cellWidth = (width - margin * 2 - gap * 4) / 5
    return Array.from({ length: count }, (_, index) => ({ x: margin + index * (cellWidth + gap), y: 18, width: cellWidth, height: height - 36 }))
  }
  if (cardCode === 'all-in') return [{ x: 18, y: 18, width: width - 36, height: height - 36 }]
  const rows = Math.ceil(count / columns)
  const cellWidth = width / columns
  const cellHeight = height / rows
  return Array.from({ length: count }, (_, index) => ({
    x: (index % columns) * cellWidth,
    y: Math.floor(index / columns) * cellHeight,
    width: cellWidth,
    height: cellHeight,
  }))
}

export function scratchLayoutFor(cardCode: string, symbolCount: number) {
  const columns = cardCode === 'eternal-color-diamond' ? 5 : symbolCount === 9 ? 3 : symbolCount > 4 ? 4 : Math.max(symbolCount, 1)
  const rows = Math.ceil(symbolCount / columns)
  const stageHeight = cardCode === 'eternal-color-diamond' ? 190 : cardCode === 'all-in' ? 720 : rows <= 1 ? 220 : rows * 145
  return {
    columns,
    stageHeight,
    cells: getCellBounds(cardCode, symbolCount, columns, 720, stageHeight),
  }
}

export function ScratchCard({ cardCode, cardName, symbols, scratchLevel, prizeTier = 'none', onComplete, onProgress }: ScratchCardProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const debrisLayerRef = useRef<HTMLDivElement>(null)
  const scratchToolRef = useRef<HTMLDivElement>(null)
  const opaqueIndexesRef = useRef<number[][]>([])
  const revealedRef = useRef<Set<number>>(new Set())
  const drawingRef = useRef(false)
  const completedRef = useRef(false)
  const lastPointRef = useRef<{ x: number; y: number } | null>(null)
  const moveCountRef = useRef(0)
  const [progress, setProgress] = useState(0)
  const [revealedIndexes, setRevealedIndexes] = useState<number[]>([])
  const [isScratching, setIsScratching] = useState(false)
  const { columns, stageHeight, cells: cellBounds } = scratchLayoutFor(cardCode, symbols.length)
  const revealThreshold = cardCode === 'all-in' ? 70 : 65
  const rangePercent = scratchEffects[Math.max(1, Math.min(10, scratchLevel))]
  const brushRadius = Math.max(6, stageHeight * rangePercent / 200)
  const tripleMatch = symbols.length === 3 && new Set(symbols).size === 1

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ratio = Math.min(window.devicePixelRatio || 1, 2)
    const width = 720
    const height = stageHeight
    canvas.width = width * ratio
    canvas.height = height * ratio
    const context = canvas.getContext('2d', { willReadFrequently: true })
    if (!context) return
    completedRef.current = false
    moveCountRef.current = 0
    opaqueIndexesRef.current = []
    revealedRef.current = new Set()
    setRevealedIndexes([])
    setProgress(0)
    context.scale(ratio, ratio)
    const gradient = context.createLinearGradient(0, 0, width, height)
    gradient.addColorStop(0, '#85898a')
    gradient.addColorStop(0.22, '#c8c9c5')
    gradient.addColorStop(0.48, '#9b9e9d')
    gradient.addColorStop(0.72, '#dad7ce')
    gradient.addColorStop(1, '#777b7d')
    context.fillStyle = gradient
    context.textAlign = 'center'
    if (cardCode === 'eternal-color-diamond') {
      const gap = 12
      const margin = 18
      const cellWidth = (width - margin * 2 - gap * 4) / 5
      diamondAppraisals.forEach((_, index) => {
        const x = margin + index * (cellWidth + gap)
        context.beginPath()
        context.roundRect(x, 18, cellWidth, height - 36, 13)
        context.fill()
        context.fillStyle = gradient
      })
    } else if (cardCode === 'all-in') {
      context.beginPath()
      context.arc(width / 2, height / 2, Math.min(width, height) / 2 - 18, 0, Math.PI * 2)
      context.fill()
    } else if (cardCode === 'street-store' || cardCode === 'arcade-challenge' || cardCode === 'gold-mine' || cardCode === 'rocket-launch') {
      const coatingCells = getCellBounds(cardCode, symbols.length, columns, width, height)
      coatingCells.forEach((cell) => {
        context.beginPath()
        context.roundRect(cell.x + 4, cell.y + 4, cell.width - 8, cell.height - 8, Math.min(16, cell.height * .1))
        context.fill()
      })
    } else if (cardCode === 'deep-sea-salvage') {
      const coatingCells = getCellBounds(cardCode, symbols.length, columns, width, height)
      coatingCells.forEach((cell) => {
        context.beginPath()
        context.ellipse(cell.x + cell.width / 2, cell.y + cell.height / 2, cell.width * .43, cell.height * .43, 0, 0, Math.PI * 2)
        context.fill()
      })
    } else {
      context.fillRect(0, 0, width, height)
    }
    context.save()
    context.globalCompositeOperation = 'source-atop'
    context.lineWidth = 1
    for (let offset = -height; offset < width + height; offset += 19) {
      context.strokeStyle = offset % 38 === 0 ? 'rgba(255,255,255,.18)' : 'rgba(42,46,47,.12)'
      context.beginPath()
      context.moveTo(offset, height)
      context.lineTo(offset + height, 0)
      context.stroke()
    }
    for (let index = 0; index < 86; index++) {
      const x = (index * 83 + 17) % width
      const y = (index * 47 + 29) % height
      const radius = 1 + (index % 3) * .65
      context.fillStyle = index % 2 ? 'rgba(245,242,230,.2)' : 'rgba(45,47,48,.18)'
      context.beginPath()
      context.arc(x, y, radius, 0, Math.PI * 2)
      context.fill()
    }
    if (cardCode === 'lingqian-ticket') {
      context.fillStyle = 'rgba(45,48,47,.16)'
      context.font = '800 40px Georgia, serif'
      for (let x = 78; x < width; x += 175) context.fillText('$', x, height / 2 + 14)
    }
    context.restore()
    const cells = getCellBounds(cardCode, symbols.length, columns, width, height)
    const initialPixels = context.getImageData(0, 0, canvas.width, canvas.height).data
    opaqueIndexesRef.current = cells.map((cell) => {
      const indexes: number[] = []
      for (let y = cell.y + 5; y < cell.y + cell.height - 4; y += 10) {
        for (let x = cell.x + 5; x < cell.x + cell.width - 4; x += 10) {
          const pixelX = Math.min(canvas.width - 1, Math.floor(x * ratio))
          const pixelY = Math.min(canvas.height - 1, Math.floor(y * ratio))
          const pixelIndex = (pixelY * canvas.width + pixelX) * 4 + 3
          if (initialPixels[pixelIndex] >= 64) indexes.push(pixelIndex)
        }
      }
      return indexes
    })
    context.globalCompositeOperation = 'destination-out'
  }, [cardCode, columns, stageHeight, symbols.length])

  function scratch(event: React.PointerEvent<HTMLCanvasElement>) {
    if (!drawingRef.current || completedRef.current) return
    const canvas = canvasRef.current
    const context = canvas?.getContext('2d', { willReadFrequently: true })
    if (!canvas || !context) return
    const bounds = canvas.getBoundingClientRect()
    const x = (event.clientX - bounds.left) * (720 / bounds.width)
    const y = (event.clientY - bounds.top) * (stageHeight / bounds.height)
    const last = lastPointRef.current ?? { x, y }
    context.beginPath()
    context.lineCap = 'round'
    context.lineJoin = 'round'
    context.lineWidth = brushRadius * 2
    context.moveTo(last.x, last.y)
    context.lineTo(x, y)
    context.stroke()
    const deltaX = x - last.x
    const deltaY = y - last.y
    const distance = Math.hypot(deltaX, deltaY)
    if (distance > 1) {
      context.save()
      context.globalAlpha = .72
      context.lineWidth = brushRadius * .52
      const normalX = -deltaY / distance
      const normalY = deltaX / distance
      const edgeOffset = brushRadius * .36
      context.beginPath()
      context.moveTo(last.x + normalX * edgeOffset, last.y + normalY * edgeOffset)
      context.lineTo(x + normalX * edgeOffset, y + normalY * edgeOffset)
      context.stroke()
      context.beginPath()
      context.moveTo(last.x - normalX * edgeOffset, last.y - normalY * edgeOffset)
      context.lineTo(x - normalX * edgeOffset, y - normalY * edgeOffset)
      context.stroke()
      context.restore()
    }
    lastPointRef.current = { x, y }
    moveCountRef.current++
    updateScratchTool(x, y, Math.atan2(deltaY, deltaX))
    if (moveCountRef.current % 2 === 0) spawnScratchDebris(x, y, Math.atan2(deltaY, deltaX), 3)
    if (moveCountRef.current % 4 === 0) measureProgress(context, canvas)
  }

  function updateScratchTool(x: number, y: number, angle: number) {
    const tool = scratchToolRef.current
    if (!tool) return
    tool.style.left = `${x / 7.2}%`
    tool.style.top = `${y / stageHeight * 100}%`
    tool.style.setProperty('--scratch-angle', `${angle}rad`)
  }

  function spawnScratchDebris(x: number, y: number, angle: number, amount: number) {
    const layer = debrisLayerRef.current
    if (!layer) return
    for (let index = 0; index < amount; index++) {
      const flake = document.createElement('i')
      const spread = (Math.random() - .5) * Math.PI * .9
      const force = 13 + Math.random() * 27
      const direction = angle - Math.PI / 2 + spread
      flake.className = `scratch-flake tone-${(moveCountRef.current + index) % 3}`
      flake.style.left = `${x / 7.2}%`
      flake.style.top = `${y / stageHeight * 100}%`
      flake.style.width = `${2.5 + Math.random() * 5}px`
      flake.style.height = `${1.5 + Math.random() * 3}px`
      flake.style.setProperty('--flake-x', `${Math.cos(direction) * force}px`)
      flake.style.setProperty('--flake-y', `${Math.sin(direction) * force - 8}px`)
      flake.style.setProperty('--flake-spin', `${(Math.random() - .5) * 540}deg`)
      flake.addEventListener('animationend', () => flake.remove(), { once: true })
      layer.appendChild(flake)
    }
  }

  function measureProgress(context: CanvasRenderingContext2D, canvas: HTMLCanvasElement) {
    const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data
    let totalTransparent = 0
    let totalSampled = 0
    const newlyRevealed: number[] = []
    opaqueIndexesRef.current.forEach((indexes, cellIndex) => {
      const transparent = indexes.reduce((count, pixelIndex) => count + (pixels[pixelIndex] < 64 ? 1 : 0), 0)
      totalTransparent += transparent
      totalSampled += indexes.length
      const cellProgress = indexes.length ? (transparent / indexes.length) * 100 : 0
      // The prize artwork exists below the coating from the first frame. Mark the
      // cell as revealed after the first meaningful scratch so its entrance
      // effect never waits for the whole ticket to complete.
      if (cellProgress >= 4 && !revealedRef.current.has(cellIndex)) {
        revealedRef.current.add(cellIndex)
        newlyRevealed.push(cellIndex)
      }
    })
    if (newlyRevealed.length) setRevealedIndexes([...revealedRef.current].sort((a, b) => a - b))
    const nextProgress = totalSampled ? Math.round((totalTransparent / totalSampled) * 100) : 0
    setProgress(nextProgress)
    onProgress?.(nextProgress)
    if (nextProgress >= revealThreshold) finish(context, canvas)
  }

  function finish(context: CanvasRenderingContext2D, canvas: HTMLCanvasElement) {
    if (completedRef.current) return
    completedRef.current = true
    const burstCells = cellBounds.length ? cellBounds : [{ x: 0, y: 0, width: 720, height: stageHeight }]
    burstCells.forEach((cell, index) => {
      if (index % Math.max(1, Math.ceil(burstCells.length / 6)) === 0) {
        spawnScratchDebris(cell.x + cell.width / 2, cell.y + cell.height / 2, -Math.PI / 2, 5)
      }
    })
    context.clearRect(0, 0, 720, stageHeight)
    setRevealedIndexes(symbols.map((_, index) => index))
    setProgress(100)
    onProgress?.(100)
    onComplete()
  }

  const complete = completedRef.current
  return (
    <div className={`scratch-card scratch-card-${cardCode} tier-${prizeTier} ${tripleMatch ? 'is-triple' : ''} ${isScratching ? 'is-scratching' : ''} ${complete ? 'is-complete' : ''}`}>
      <div className="ticket-heading">
          <span>{cardName}</span>
          <strong>{cardCode === 'eternal-color-diamond' ? '五项彩钻鉴定' : cardCode === 'all-in' ? '唯一终局刮层' : `${symbols.length} 格刮奖区域`}</strong>
        </div>
      <div className="scratch-stage" style={{ aspectRatio: `720 / ${stageHeight}` }}>
        <div className="symbols" aria-label={`${cardName}刮奖结果`} style={{ display: 'block' }}>
          {symbols.map((symbol, index) => {
            const visual = getSymbolVisual(symbol)
            const revealed = revealedIndexes.includes(index)
            const stateClasses = symbolStateClassNames(cardCode, symbols, index, complete)
            const cell = cellBounds[index]
            return (
              <div
                className={`symbol symbol-${symbolClassName(symbol)} ${revealed ? 'revealed' : ''} ${stateClasses}`}
                key={`${symbol}-${index}`}
                aria-label={visual.label}
                style={{ position: 'absolute', left: `${cell.x / 7.2}%`, top: `${cell.y / stageHeight * 100}%`, width: `${cell.width / 7.2}%`, height: `${cell.height / stageHeight * 100}%` }}
              >
                <TicketSymbolGlyph cardCode={cardCode} symbol={symbol} />
                <small>{symbol.startsWith('目标·') ? `目标：${visual.label}` : visual.label}</small>
              </div>
            )
          })}
        </div>
        <div className="scratch-debris-layer" ref={debrisLayerRef} aria-hidden="true" />
        <div className="scratch-tool" ref={scratchToolRef} aria-hidden="true"><i /></div>
        <div className="scratch-finish-flash" aria-hidden="true" />
        <canvas
          ref={canvasRef}
          className="scratch-layer"
          onPointerDown={(event) => {
            drawingRef.current = true
            setIsScratching(true)
            lastPointRef.current = null
            event.currentTarget.setPointerCapture(event.pointerId)
            scratch(event)
          }}
          onPointerMove={scratch}
          onPointerUp={(event) => {
            drawingRef.current = false
            setIsScratching(false)
            lastPointRef.current = null
            const context = event.currentTarget.getContext('2d', { willReadFrequently: true })
            if (context && !completedRef.current) measureProgress(context, event.currentTarget)
          }}
          onPointerCancel={() => { drawingRef.current = false; lastPointRef.current = null; setIsScratching(false) }}
          aria-label="刮奖区域"
        />
      </div>
      <div className="scratch-progress" aria-live="polite">
        <span style={{ width: `${progress}%` }} />
      </div>
      <p>{cardCode === 'all-in' ? '好运道具无效 · 机器人禁用 · ' : `刮片 Lv.${scratchLevel} · 单次完整划动参考覆盖 ${rangePercent}% · `}刮开{revealThreshold}%后揭晓</p>
    </div>
  )
}
