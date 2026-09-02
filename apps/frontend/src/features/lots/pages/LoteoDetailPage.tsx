import { ArrowLeft } from 'lucide-react'
import { Link, useParams } from 'react-router'

export default function LoteoDetailPage() {
  const { loteoId } = useParams()

  return (
    <section className="flex flex-col gap-4">
      <Link
        to="/lotes"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft aria-hidden className="size-4" />
        Volver al listado
      </Link>
      <h1 className="text-2xl font-semibold">Loteo {loteoId}</h1>
      <p className="text-muted-foreground">
        El detalle del loteo estará disponible próximamente.
      </p>
    </section>
  )
}
