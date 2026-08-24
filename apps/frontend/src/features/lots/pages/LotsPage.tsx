import LoteoDataCard from '../components/LoteoDataCard'
import LoteoPageHeader from '../components/LoteoPageHeader'
import LoteoPlanCard from '../components/LoteoPlanCard'
import { useDxfPlan } from '../hooks/use-dxf-plan'
import { useLoteoFields } from '../hooks/use-loteo-fields'

export default function LotsPage() {
  const fields = useLoteoFields()
  const plan = useDxfPlan()

  function handleDiscard() {
    fields.reset()
    plan.reset()
  }

  return (
    <section className="flex min-h-0 flex-1 flex-col gap-4">
      <LoteoPageHeader hasPlan={plan.hasPlan} onDiscard={handleDiscard} />

      <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 md:grid-cols-2">
        <LoteoDataCard
          className="md:self-start"
          values={fields.values}
          onChange={fields.onChange}
        />
        <LoteoPlanCard
          className="md:h-full"
          hasPlan={plan.hasPlan}
          fileName={plan.fileName}
          error={plan.error}
          issues={plan.issues}
          polygons={plan.polygons}
          visibleLayers={plan.visibleLayers}
          onVisibleLayersChange={plan.onVisibleLayersChange}
          onParsed={plan.onParsed}
          onError={plan.onError}
          onCleared={plan.onCleared}
        />
      </div>
    </section>
  )
}
