import { useEffect, useRef, useState } from 'react'

type ScratchCardProps = {
  cardName: string
  symbols: string[]
  onComplete: () => void
}

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
}

export function ScratchCard({ cardName, symbols, onComplete }: ScratchCardProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const drawingRef = useRef(false)
  const completedRef = useRef(false)
  const [progress, setProgress] = useState(0)
  const columns = symbols.length === 9 ? 3 : symbols.length > 4 ? 4 : Math.max(symbols.length, 1)
  const rows = Math.ceil(symbols.length / columns)
  const stageHeight = rows <= 1 ? 220 : rows * 145

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
    setProgress(0)
    context.scale(ratio, ratio)
    const gradient = context.createLinearGradient(0, 0, width, height)
    gradient.addColorStop(0, '#d9d7ce')
    gradient.addColorStop(0.45, '#a7a8a5')
    gradient.addColorStop(1, '#e7e2d4')
    context.fillStyle = gradient
    context.fillRect(0, 0, width, height)
    context.fillStyle = 'rgba(60, 65, 66, .38)'
    context.font = '700 26px system-ui'
    context.textAlign = 'center'
    context.fillText('按住并刮开', width / 2, height / 2 + 8)
    context.globalCompositeOperation = 'destination-out'
  }, [stageHeight])

  function scratch(event: React.PointerEvent<HTMLCanvasElement>) {
    if (!drawingRef.current || completedRef.current) return
    const canvas = canvasRef.current
    const context = canvas?.getContext('2d', { willReadFrequently: true })
    if (!canvas || !context) return
    const bounds = canvas.getBoundingClientRect()
    const x = (event.clientX - bounds.left) * (720 / bounds.width)
    const y = (event.clientY - bounds.top) * (stageHeight / bounds.height)
    context.beginPath()
    context.arc(x, y, 30, 0, Math.PI * 2)
    context.fill()
    if (event.timeStamp % 4 < 1) measureProgress(context, canvas)
  }

  function measureProgress(context: CanvasRenderingContext2D, canvas: HTMLCanvasElement) {
    const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data
    let transparent = 0
    let sampled = 0
    for (let index = 3; index < pixels.length; index += 4 * 24) {
      sampled++
      if (pixels[index] < 64) transparent++
    }
    const nextProgress = Math.round((transparent / sampled) * 100)
    setProgress(nextProgress)
    if (nextProgress >= 45) finish(context, canvas)
  }

  function finish(context: CanvasRenderingContext2D, canvas: HTMLCanvasElement) {
    if (completedRef.current) return
    completedRef.current = true
    context.clearRect(0, 0, canvas.width, canvas.height)
    setProgress(100)
    onComplete()
  }

  return (
    <div className="scratch-card">
      <div className="ticket-heading">
          <span>{cardName}</span>
          <strong>{symbols.length} 格刮奖区域</strong>
        </div>
      <div className="scratch-stage" style={{ aspectRatio: `720 / ${stageHeight}` }}>
        <div className="symbols" aria-hidden={!completedRef.current} style={{ gridTemplateColumns: `repeat(${columns}, 1fr)` }}>
          {symbols.map((symbol, index) => {
            const baseSymbol = symbol.replace(/^目标·/, '')
            const fuelValue = symbol.match(/^燃料 (\d)$/)?.[1]
            const visual = fuelValue
              ? { emoji: `⛽${fuelValue}`, label: symbol }
              : symbolVisuals[baseSymbol] ?? { emoji: '✦', label: symbol }
            return (
              <div className="symbol" key={`${symbol}-${index}`}>
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
            event.currentTarget.setPointerCapture(event.pointerId)
            scratch(event)
          }}
          onPointerMove={scratch}
          onPointerUp={() => { drawingRef.current = false }}
          onPointerCancel={() => { drawingRef.current = false }}
          aria-label="刮奖区域"
        />
      </div>
      <div className="scratch-progress" aria-live="polite">
        <span style={{ width: `${progress}%` }} />
      </div>
      <p>刮开45%后自动揭晓完整结果</p>
    </div>
  )
}
