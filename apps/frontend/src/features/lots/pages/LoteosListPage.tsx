import { useState } from 'react'
import { Plus } from 'lucide-react'
import { Link } from 'react-router'
import { Alert, AlertDescription, AlertTitle } from '../../../shared/ui/alert'
import { Button } from '../../../shared/ui/button'
import { Input } from '../../../shared/ui/input'
import { Label } from '../../../shared/ui/label'
import { useLoteos } from '../hooks/use-loteos'
import LoteoZocaloCard from '../components/LoteoZocaloCard'

type LoteosListPageProps = {
  accessToken: string | null
}

export default function LoteosListPage({ accessToken }: LoteosListPageProps) {
  const [search, setSearch] = useState('')
  const { loteos, isLoading, error } = useLoteos(accessToken ?? '', search)

  const isSearching = search.trim() !== ''
  const showSearch = loteos.length > 0 || isSearching

  return (
    <section className="flex flex-col gap-4">
      <header className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-semibold">Loteos</h1>
            <span className="text-sm text-muted-foreground">{loteos.length}</span>
          </div>
          <p className="text-sm text-muted-foreground">
            El listado de loteos cargados. Tocá uno para ver el plano y los lotes.
          </p>
        </div>
        <Button render={<Link to="/lotes/nuevo" />}>
          <Plus aria-hidden />
          Nuevo loteo
        </Button>
      </header>

      {showSearch && (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="buscar-loteo">Buscar</Label>
          <Input
            id="buscar-loteo"
            type="search"
            placeholder="Nombre o ubicación"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className="sm:max-w-xs"
          />
        </div>
      )}

      {isLoading && loteos.length === 0 && (
        <p className="text-muted-foreground">Cargando loteos…</p>
      )}

      {error && (
        <Alert variant="destructive">
          <AlertTitle>No se pudo cargar el listado</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {!isLoading && !error && loteos.length === 0 && !isSearching && (
        <p className="text-muted-foreground">Todavía no hay loteos cargados.</p>
      )}

      {!isLoading && !error && loteos.length === 0 && isSearching && (
        <p className="text-muted-foreground">No se encontraron loteos con esa búsqueda.</p>
      )}

      {loteos.length > 0 && (
        <ul className="flex flex-col gap-3">
          {loteos.map((loteo) => (
            <li key={loteo.id}>
              <LoteoZocaloCard loteo={loteo} />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
