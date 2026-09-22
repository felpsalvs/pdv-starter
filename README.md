# Soparia PDV

Sistema de ponto de venda simples para a soparia. Roda no próprio computador
do estabelecimento, funciona offline no dia a dia, e não precisa de nenhuma
assinatura ou serviço pago.

## O que você precisa antes de começar

- O computador da soparia (Windows).
- Internet **de vez em quando** — só pra checar se tem uma versão nova (o
  sistema faz isso sozinho ao abrir; se não tiver internet, ele
  simplesmente continua com a versão que já está instalada).
- Nada mais. Não precisa instalar Node, nem nenhum outro programa.

## Como instalar (fazer uma vez só)

1. Baixe o arquivo `pdv.exe` da
   [última versão publicada](https://github.com/felpsalvs/pdv-starter/releases/latest)
   (em "Assets", o arquivo `pdv_windows_amd64.zip` — extraia o zip, o
   `pdv.exe` está dentro).
2. Coloque o `pdv.exe` numa pasta só dele no computador da soparia (ex:
   `C:\SopariaPDV`). É nessa pasta que o sistema vai guardar os pedidos, o
   cardápio e os backups — depois de instalar, não mova esse arquivo pra
   outro lugar sem mover a pasta inteira junto.
3. Dê dois cliques no `pdv.exe` pra testar. O navegador deve abrir sozinho
   na tela do sistema.

Dica: crie um atalho do `pdv.exe` na área de trabalho, pra não precisar
abrir a pasta toda vez.

## Como usar todo dia

1. Dê dois cliques no `pdv.exe` (ou no atalho da área de trabalho).
2. Uma janela preta abre rapidamente — ela verifica se tem uma versão nova
   (e atualiza sozinha, se tiver) e liga o sistema.
3. O navegador abre sozinho na tela do sistema. Se não abrir, acesse
   `http://localhost:3000` manualmente.
4. Deixe essa janela preta aberta enquanto estiver usando o sistema —
   fechar ela desliga o programa (ele salva um backup automaticamente
   antes de fechar, então pode fechar sem medo no fim do dia).

### Atualizações

Você não precisa fazer nada: toda vez que o `pdv.exe` é aberto, ele
confere sozinho se existe uma versão mais nova publicada e, se existir,
baixa e aplica antes de abrir o sistema (fazendo um backup do banco antes,
por segurança). Se não tiver internet no momento, ele simplesmente abre
com a versão que já está instalada — nada trava.

## Migrando da versão antiga (Node)

Se você já usava a versão antiga (que precisava de `npm install`/`npm
start`), seus dados não se perdem:

1. Feche o sistema antigo (`Ctrl+C` no terminal, se ainda estiver aberto).
2. Instale o `pdv.exe` numa pasta nova, como descrito acima.
3. Copie o arquivo `data/pdv.db` da pasta do projeto antigo para dentro de
   `data/pdv.db` na pasta nova do `pdv.exe` (crie a pasta `data` se ela
   ainda não existir).
4. Se você tinha configurado a impressora, copie também o
   `printer.config.json` da pasta antiga para a pasta nova.
5. Abra o `pdv.exe` normalmente — na primeira vez ele reconhece o banco
   antigo e segue de onde parou, sem perder nenhum pedido ou fechamento de
   caixa.

## Configurando a impressora da cozinha

O sistema manda os bytes da impressão direto pro Windows imprimir — não
precisa instalar nenhum programa extra nem compilador pra isso funcionar.
Só precisa de um passo único de configuração:

1. Ligue a impressora térmica por USB e deixe o Windows instalar o driver
   dela normalmente (geralmente reconhece sozinho; se não, instale o driver
   que veio com a impressora, ou use o driver genérico "Generic / Text
   Only" do próprio Windows).
2. Abra **Configurações → Dispositivos → Impressoras e scanners**, clique
   na impressora térmica → **Propriedades da impressora → Compartilhamento**,
   e marque **"Compartilhar esta impressora"**. Anote o **nome do
   compartilhamento** que você definir ali (pode ser o mesmo nome da
   impressora, sem espaços).
3. Na pasta do projeto, copie o arquivo `printer.config.example.json`
   e renomeie a cópia para `printer.config.json`.
4. Abra `printer.config.json` num editor de texto (Bloco de Notas serve) e
   troque o texto pelo **nome do compartilhamento** do passo 2:
   ```json
   {
     "printerName": "POS-58"
   }
   ```
5. Salve o arquivo e reinicie o sistema (feche a janela preta e abra o
   `pdv.exe` de novo).

O passo 2 (compartilhar) é necessário porque é assim que o sistema entrega
os bytes da impressão pro Windows, sem precisar de nenhuma biblioteca
nativa instalada — só esse compartilhamento local, que não expõe a
impressora pra rede nenhuma de fora do próprio computador.

**Se a impressora estiver desligada ou não configurada, o sistema continua
funcionando normalmente** — o pedido é salvo e a tela avisa que não foi
possível imprimir, mas nada trava.

## Telas do sistema

- **Balcão**: tela principal para lançar os pedidos do dia a dia — pensada para
  usar só o teclado, sem precisar de mouse. Veja "Atalhos do balcão" abaixo.
- **Dia**: lista todos os pedidos do dia — contas de mesa em aberto, pagos e
  cancelados. É onde você recebe o pagamento de uma mesa, reimprime um ticket
  ou cancela um pedido errado.
- **Caixa**: abrir o caixa no início do dia, registrar sangria (retirada) ou
  suprimento (reforço) durante o turno, e fechar no final conferindo o valor
  esperado **na gaveta** (só dinheiro) com o valor contado.
- **Cardápio**: organizar produtos em categorias, marcar quais estão
  "disponíveis hoje" (a sopa do dia), editar preço/nome e reativar itens
  removidos por engano.

## Atalhos do balcão

A tela de Balcão foi feita pra ser usada só com o teclado — o cursor já
começa dentro do campo de busca.

- Digite o nome de uma sopa e aperte **Enter** para adicionar ao pedido.
- Digite um número antes do nome (ex: `2 caldo verde`) para adicionar mais de
  uma unidade de uma vez.
- **↑ / ↓** navegam entre os produtos encontrados na busca.
- **/** (com a busca vazia) abre o campo de observação do último item
  adicionado (ex: "sem cebola").
- **F2** define pra quem é o pedido — Balcão ou Mesa (número).
- **F4** abre a tela de pagamento — Dinheiro, Pix, Débito ou Crédito. Em
  dinheiro, o troco aparece calculado na hora.
- **F8** envia o pedido pra cozinha **sem cobrar agora** — fica como conta em
  aberto (útil pra mesa que só vai pagar no final).
- **F9** limpa o pedido atual.
- **Esc** limpa o campo de busca.

## As três impressões

Cada pedido pode gerar até três documentos na impressora térmica:

1. **Ticket da cozinha** — sai assim que o pedido é enviado, com a senha, a
   mesa/identificação e os itens.
2. **Etiqueta da embalagem** — uma por sopa, com a senha e o nome do item, pra
   colar na embalagem (substitui a etiqueta escrita à mão).
3. **Recibo** — sai quando o pedido é pago, com o total e a forma de
   pagamento.

Se a impressora estiver desligada, o pedido continua sendo salvo — a tela do
Balcão avisa que não foi possível imprimir, e dá pra reimprimir depois pela
tela **Dia**.

## Onde ficam os dados

Tudo é salvo num único arquivo em `data/pdv.db` — pedidos, itens, caixa,
cardápio. Nada é apagado automaticamente: o histórico fica ali guardado pra
sempre, mesmo de anos atrás. Não apague essa pasta `data`, ou você perde tudo.

### Backup automático

Sempre que o sistema está ligado, ele salva sozinho uma cópia de segurança do
banco na pasta `backups/` — uma ao abrir o sistema, uma a cada 4 horas, e uma
última ao fechar (`Ctrl+C`). Ele guarda as 30 cópias mais recentes e vai
apagando as mais antigas sozinho. Você não precisa fazer nada pra isso
acontecer.

**Isso não substitui um backup fora do computador.** Se o notebook for
perdido, roubado ou o disco quebrar, as cópias em `backups/` se perdem
junto — elas protegem contra um banco corrompido ou apagado sem querer, não
contra perder o computador inteiro. De vez em quando, copie o arquivo mais
recente da pasta `backups/` pra um pendrive ou pasta na nuvem — como já é uma
cópia pronta e consistente, basta arrastar o arquivo, sem precisar fechar o
sistema.

## Para quem for mexer no código (desenvolvimento)

O backend é Go (`cmd/pdv`, com o resto em `internal/`) e o frontend é
Svelte (`frontend/`), buildado com Vite e embutido dentro do binário Go —
por isso o frontend precisa estar buildado (`dist/`) antes de compilar ou
rodar o backend.

```
npm install              # instala as deps do frontend e já builda o dist/
                          # (rode de novo depois de qualquer mudança no frontend)
PDV_HOME=. go run ./cmd/pdv -no-update -no-browser
```

`PDV_HOME=.` faz o backend usar `./data`, `./backups` e
`./printer.config.json` relativos à pasta do projeto, em vez de relativos
a um `.exe` instalado — assim o `go run` não mexe em nada fora do repo.
`-no-update` evita que ele tente se auto-atualizar num build de
desenvolvimento (que não tem uma versão real).

Rodando `npm run dev:frontend` num terminal separado dá hot-reload do
frontend (proxying `/api` pro backend Go na porta 3000).

Testes: `go test ./...`. Build local do `.exe` pra testar num Windows:
`GOOS=windows GOARCH=amd64 go build -o pdv.exe ./cmd/pdv`.
