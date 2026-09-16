import { useEffect, useRef, useState } from 'react'

type ScratchCardProps = {
  cardCode: string
  cardName: string
  symbols: string[]
  scratchLevel: number
  prizeTier?: string
  onComplete: () => void
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
  if (cardCode === 'eternal-color-diamond') {
    const gap = 12
    const margin = 18
    const cellWidth = (width - margin * 2 - gap * 4) / 5
    return Array.from({ length: count }, (_, index) => ({ x: margin + index * (cellWidth + gap), y: 18, width: cellWidth, height: height - 36 }))
  }
  if (cardCode === 'all-in') return [{ x: width / 2 - 105, y: height / 2 - 105, width: 210, height: 210 }]
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

export function ScratchCard({ cardCode, cardName, symbols, scratchLevel, prizeTier = 'none', onComplete }: ScratchCardProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const opaqueIndexesRef = useRef<number[][]>([])
  const revealedRef = useRef<Set<number>>(new Set())
  const drawingRef = useRef(false)
  const completedRef = useRef(false)
  const lastPointRef = useRef<{ x: number; y: number } | null>(null)
  const moveCountRef = useRef(0)
  const [progress, setProgress] = useState(0)
  const [revealedIndexes, setRevealedIndexes] = useState<number[]>([])
  const columns = cardCode === 'eternal-color-diamond' ? 5 : symbols.length === 9 ? 3 : symbols.length > 4 ? 4 : Math.max(symbols.length, 1)
  const rows = Math.ceil(symbols.length / columns)
  const stageHeight = cardCode === 'eternal-color-diamond' ? 190 : cardCode === 'all-in' ? 270 : rows <= 1 ? 220 : rows * 145
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
    gradient.addColorStop(0, '#d9d7ce')
    gradient.addColorStop(0.45, '#a7a8a5')
    gradient.addColorStop(1, '#e7e2d4')
    context.fillStyle = gradient
    context.textAlign = 'center'
    if (cardCode === 'eternal-color-diamond') {
      const gap = 12
      const margin = 18
      const cellWidth = (width - margin * 2 - gap * 4) / 5
      diamondAppraisals.forEach((label, index) => {
        const x = margin + index * (cellWidth + gap)
        context.beginPath()
        context.roundRect(x, 18, cellWidth, height - 36, 13)
        context.fill()
        context.fillStyle = 'rgba(53, 55, 72, .58)'
        context.font = '700 18px system-ui'
        context.fillText(label, x + cellWidth / 2, height / 2 + 6)
        context.fillStyle = gradient
      })
    } else if (cardCode === 'all-in') {
      context.beginPath()
      context.arc(width / 2, height / 2, 105, 0, Math.PI * 2)
      context.fill()
      context.fillStyle = 'rgba(44, 26, 25, .68)'
      context.font = '800 24px system-ui'
      context.fillText('最终抉择', width / 2, height / 2 + 8)
    } else {
      context.fillRect(0, 0, width, height)
      context.fillStyle = 'rgba(60, 65, 66, .38)'
      context.font = '700 26px system-ui'
      context.fillText('按住并刮开', width / 2, height / 2 + 8)
    }
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
    lastPointRef.current = { x, y }
    moveCountRef.current++
    if (moveCountRef.current % 4 === 0) measureProgress(context, canvas)
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
      if (cellProgress >= revealThreshold && !revealedRef.current.has(cellIndex)) {
        revealedRef.current.add(cellIndex)
        newlyRevealed.push(cellIndex)
      }
    })
    if (newlyRevealed.length) setRevealedIndexes([...revealedRef.current].sort((a, b) => a - b))
    const nextProgress = totalSampled ? Math.round((totalTransparent / totalSampled) * 100) : 0
    setProgress(nextProgress)
    if (nextProgress >= revealThreshold) finish(context, canvas)
  }

  function finish(context: CanvasRenderingContext2D, canvas: HTMLCanvasElement) {
    if (completedRef.current) return
    completedRef.current = true
    context.clearRect(0, 0, 720, stageHeight)
    setRevealedIndexes(symbols.map((_, index) => index))
    setProgress(100)
    onComplete()
  }

  const complete = completedRef.current
  return (
    <div className={`scratch-card scratch-card-${cardCode} tier-${prizeTier} ${tripleMatch ? 'is-triple' : ''} ${complete ? 'is-complete' : ''}`}>
      <div className="ticket-heading">
          <span>{cardName}</span>
          <strong>{cardCode === 'eternal-color-diamond' ? '五项彩钻鉴定' : cardCode === 'all-in' ? '唯一终局刮层' : `${symbols.length} 格刮奖区域`}</strong>
        </div>
      <div className="scratch-stage" style={{ aspectRatio: `720 / ${stageHeight}` }}>
        <div className="symbols" aria-hidden={!complete} style={{ gridTemplateColumns: `repeat(${columns}, 1fr)` }}>
          {symbols.map((symbol, index) => {
            const visual = getSymbolVisual(symbol)
            const revealed = revealedIndexes.includes(index)
            const stateClasses = symbolStateClassNames(cardCode, symbols, index, complete)
            return (
              <div className={`symbol symbol-${symbolClassName(symbol)} ${revealed ? 'revealed' : ''} ${stateClasses}`} key={`${symbol}-${index}`} aria-label={visual.label}>
                <span>{visual.emoji}</span>
                <small>{symbol.startsWith('目标·') ? `目标：${visual.label}` : visual.label}</small>
              </div>
            )
          })}
        </div>
        <canvas
          ref={canvasRef}
          className="scratch-layer"
          onPointerDown={(event) => {
            drawingRef.current = true
            lastPointRef.current = null
            event.currentTarget.setPointerCapture(event.pointerId)
            scratch(event)
          }}
          onPointerMove={scratch}
          onPointerUp={(event) => {
            drawingRef.current = false
            lastPointRef.current = null
            const context = event.currentTarget.getContext('2d', { willReadFrequently: true })
            if (context && !completedRef.current) measureProgress(context, event.currentTarget)
          }}
          onPointerCancel={() => { drawingRef.current = false; lastPointRef.current = null }}
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
