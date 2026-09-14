package main
import("context";"encoding/json";"fmt";"net/url";"os";"strings";"time";"github.com/jackc/pgx/v5";"github.com/jackc/pgx/v5/pgxpool";"loteosapp/backend/internal/business/gateway";"loteosapp/backend/internal/infrastructure/repository/postgres")
func must(err error){if err!=nil{panic(err)}}
func id(n int)string{return fmt.Sprintf("90000000-0000-0000-0000-%012d",n)}
type gate struct{ready chan struct{}}
func(g *gate)TraceQueryStart(ctx context.Context,_ *pgx.Conn,d pgx.TraceQueryStartData)context.Context{if strings.Contains(d.SQL,"SELECT true FROM clientes")&&strings.Contains(d.SQL,"fecha_baja IS NULL"){select{case g.ready<-struct{}{}:default:}};return ctx}
func(g *gate)TraceQueryEnd(context.Context,*pgx.Conn,pgx.TraceQueryEndData){}
func main(){
 ctx,cancel:=context.WithTimeout(context.Background(),time.Minute);defer cancel()
 schema:=os.Args[1];if !strings.HasPrefix(schema,"review_reservations_"){panic("invalid isolated schema")}
 dsn,err:=url.Parse(os.Getenv("DATABASE_URL"));must(err);q:=dsn.Query();q.Set("search_path",schema+",extensions");q.Set("application_name","reservation-review-probes");dsn.RawQuery=q.Encode()
 ready:=make(chan struct{},1);cfg,err:=pgxpool.ParseConfig(dsn.String());must(err);cfg.ConnConfig.Tracer=&gate{ready};cfg.MaxConns=4;pool,err:=pgxpool.NewWithConfig(ctx,cfg);must(err);defer pool.Close();repo:=postgres.NewReservationRepository(pool)
 var currentSchema string;must(pool.QueryRow(ctx,"SELECT current_schema()").Scan(&currentSchema));if currentSchema!=schema{panic("schema missing")}
 if len(os.Args)>2&&os.Args[2]=="variants"{b,err:=os.ReadFile("../../.codex/reviews/reservations-20260907/query-measurements.json");must(err);var samples []struct{SQL string};must(json.Unmarshal(b,&samples));var query string;for _,s:=range samples{if strings.Contains(s.SQL,"count(*) OVER()"){query=s.SQL}};if query==""{panic("missing query")};result:=map[string]any{};for _,name:=range []string{"admin","agency","development","search-original","search-diagnostic-fixed"}{var actor any;byAgency:=false;development:="";search:="";if name=="agency"{actor=id(6);byAgency=true};if name=="development"{development=id(1)};sql:=query;if strings.HasPrefix(name,"search"){search="Review"};if name=="search-diagnostic-fixed"{sql=strings.ReplaceAll(sql,`ESCAPE '\\'`,`ESCAPE '\'`)};var plan []byte;err=pool.QueryRow(ctx,"EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) "+sql,[]string{},development,search,actor,byAgency,"%"+search+"%",25,0).Scan(&plan);if err!=nil{result[name]=err.Error();fmt.Println(name,err)}else{result[name]=json.RawMessage(plan);var p []map[string]any;must(json.Unmarshal(plan,&p));fmt.Printf("LIST PLAN variant=%s execution_ms=%v planning_ms=%v\n",name,p[0]["Execution Time"],p[0]["Planning Time"])}};b,err=json.MarshalIndent(result,"","  ");must(err);must(os.WriteFile("../../.codex/reviews/reservations-20260907/list-variants.json",b,0644));return}
 if len(os.Args)>2&&os.Args[2]=="worker"{var lotID string;must(pool.QueryRow(ctx,"SELECT lote_id::text FROM reservas WHERE estado_actual='activa' ORDER BY fecha_vencimiento,id LIMIT 1").Scan(&lotID));_,err=pool.Exec(ctx,"UPDATE lotes SET fecha_baja=now() WHERE id=$1::uuid",lotID);must(err);defer pool.Exec(context.Background(),"UPDATE lotes SET fecha_baja=null WHERE id=$1::uuid",lotID);for i:=0;i<2;i++{r,err:=repo.ExpireDue(ctx,time.Now().Add(361*time.Hour),1);fmt.Printf("WORKER BLOCKED CANDIDATE run=%d candidates=%d processed=%d skipped=%d failures=%d error=%v\n",i+1,r.Candidates,r.Processed,r.Skipped,len(r.Failures),err)};return}
 if len(os.Args)>2&&os.Args[2]=="status"{rows,err:=pool.Query(ctx,`SELECT state,COALESCE(wait_event_type,''),COALESCE(wait_event,''),COALESCE(extract(epoch from now()-xact_start),0)::float8,left(query,90) FROM pg_stat_activity WHERE application_name='reservation-review' ORDER BY xact_start`);must(err);defer rows.Close();for rows.Next(){var state,kind,event,query string;var seconds float64;must(rows.Scan(&state,&kind,&event,&seconds,&query));fmt.Printf("state=%s wait=%s/%s transaction_s=%.1f query=%q\n",state,kind,event,seconds,query)};return}
 tx,err:=pool.Begin(ctx);must(err);defer tx.Rollback(context.Background())
 _,err=tx.Exec(ctx,`SELECT true FROM clientes WHERE id=$1::uuid FOR UPDATE`,id(3));must(err)
 cmd:=gateway.CreateReservationCommand{LoteoID:id(1),LoteID:id(108),ClienteID:id(3),VendedorID:id(5),ActorID:id(5),SellerIsActor:true,IdempotencyKey:"agency-race2",IdempotencyPayloadHash:strings.Repeat("a",64),CreatedAt:time.Now().UTC()}
 ch:=make(chan error,1);go func(){r,err:=repo.Create(ctx,cmd);fmt.Printf("AGENCY REVOKED WHILE CREATE WAITED state=%s error=%v\n",r.Estado,err);ch<-err}()
 select{case <-ready:case <-ctx.Done():panic("creation did not reach client lock")}
 _,err=pool.Exec(ctx,"UPDATE inmobiliarias SET fecha_baja=now() WHERE id=$1::uuid",id(4));must(err);must(tx.Commit(ctx));<-ch
 _,err=pool.Exec(ctx,"UPDATE inmobiliarias SET fecha_baja=null WHERE id=$1::uuid",id(4));must(err)
 tx,err=pool.Begin(ctx);must(err)
 sql:=`SELECT lo.estado_actual, lo.loteo_id::text FROM lotes lo JOIN loteos l ON l.id=lo.loteo_id WHERE lo.id=$1::uuid AND lo.loteo_id=$2::uuid AND lo.fecha_baja IS NULL AND l.fecha_baja IS NULL FOR UPDATE`
 _,err=tx.Exec(ctx,sql,id(109),id(1));must(err)
 other,err:=pool.Begin(ctx);must(err);_,err=other.Exec(ctx,"SET LOCAL lock_timeout='300ms'");must(err);_,err=other.Exec(ctx,sql,id(110),id(1));fmt.Printf("DIFFERENT LOT SAME DEVELOPMENT lock_error=%v\n",err);other.Rollback(context.Background());tx.Rollback(context.Background())
 _,err=pool.Exec(ctx,"UPDATE lotes SET numero=null,precio=null WHERE id=$1::uuid",id(109));must(err);cmd.LoteID=id(109);cmd.ActorID=id(2);cmd.VendedorID=id(2);cmd.SellerIsActor=false;cmd.IdempotencyKey="missing-number-price";r,err:=repo.Create(ctx,cmd);fmt.Printf("MISSING NUMBER AND PRICE state=%s error=%v\n",r.Estado,err)
 statements:=[]struct{Name,SQL string;Args []any}{
 {"agencyAssigned",`SELECT EXISTS (SELECT 1 FROM inmobiliarias a JOIN inmobiliaria_loteos il ON il.inmobiliaria_id=a.id WHERE a.id=$1::uuid AND a.fecha_baja IS NULL AND il.loteo_id=$2::uuid AND il.fecha_baja IS NULL)`,[]any{id(4),id(1)}},
 {"reconcileIdempotentCreate",`SELECT id::text,COALESCE(idempotency_payload_hash,'') FROM reservas WHERE usuario_alta=$1::uuid AND idempotency_key=$2`,[]any{id(2),"review-1"}},
 }
 output:=map[string]json.RawMessage{}
 for _,s:=range statements{var b []byte;must(pool.QueryRow(ctx,"EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+s.SQL,s.Args...).Scan(&b));output[s.Name]=b}
 b,err:=json.MarshalIndent(output,"","  ");must(err);must(os.WriteFile("../../.codex/reviews/reservations-20260907/additional-query-plans.json",b,0644))
 _=pgx.ErrNoRows
}
