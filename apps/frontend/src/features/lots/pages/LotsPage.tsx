import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import LoteoDataCard from '../components/LoteoDataCard'
import LoteoPageHeader from '../components/LoteoPageHeader'
import LoteoPlanCard from '../components/LoteoPlanCard'
import { useDxfPlan } from '../hooks/use-dxf-plan'
import { useLoteoFields } from '../hooks/use-loteo-fields'
import { useSaveLoteo } from '../hooks/use-save-loteo'

type LotsPageProps = {
  accessToken: string | null
}

export default function LotsPage({ accessToken }: LotsPageProps) {
  const navigate = useNavigate()
  const fields = useLoteoFields()
  const plan = useDxfPlan()
  const saver = useSaveLoteo(accessToken)
  const [createdLoteoId, setCreatedLoteoId] = useState<string | null>(null)

  function goToCreatedLoteo() {
    if (createdLoteoId) {
      navigate(`/lotes/${createdLoteoId}`)
    }
  }

  function handleDiscard() {
    fields.reset()
    plan.reset()
    saver.reset()
  }

  async function handleSave() {
    const outcome = await saver.save(fields.values, plan.polygons, plan.file)
    if (!outcome) {
      return
    }
    fields.reset()
    plan.reset()
    setCreatedLoteoId(outcome.loteoId)
    // A DXF-upload warning keeps the retry UI on this screen; the alta finishes
    // (and navigates) once the retry succeeds or the user continues without it.
    if (outcome.dxfWarning === null) {
      navigate(`/lotes/${outcome.loteoId}`)
    }
  }

  async function handleRetryDxf() {
    if (await saver.retryDxf()) {
      goToCreatedLoteo()
    }
  }

  return (
    <section className="flex min-h-0 flex-1 flex-col gap-4">
      <LoteoPageHeader
        hasPlan={plan.hasPlan}
        canSave={saver.status !== 'success' && fields.values.name.trim() !== ''}
        isSaving={
          saver.status === 'saving' || (saver.status === 'success' && saver.isRetryingDxf)
        }
        onSave={handleSave}
        onDiscard={handleDiscard}
      />

      {saver.status === 'error' && (
        <Alert variant="destructive">
          <AlertTitle>No se pudo guardar el loteo</AlertTitle>
          <AlertDescription>{saver.message}</AlertDescription>
        </Alert>
      )}

      {saver.status === 'success' && (
        <Alert>
          <AlertTitle>Loteo creado</AlertTitle>
          <AlertDescription>
            {saver.dxfWarning ?? 'Ya podés cargar otro loteo.'}
            {saver.dxfWarning ? (
              <>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="mt-1 min-h-11 md:h-8 md:min-h-8"
                  onClick={handleRetryDxf}
                  disabled={saver.isRetryingDxf}
                >
                  {saver.isRetryingDxf ? 'Reintentando…' : 'Reintentar carga del DXF'}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="mt-1 min-h-11 md:h-8 md:min-h-8"
                  onClick={goToCreatedLoteo}
                  disabled={saver.isRetryingDxf}
                >
                  Continuar sin el DXF
                </Button>
              </>
            ) : (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="mt-1 min-h-11 md:h-8 md:min-h-8"
                onClick={saver.reset}
              >
                Cargar otro loteo
              </Button>
            )}
          </AlertDescription>
        </Alert>
      )}

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
