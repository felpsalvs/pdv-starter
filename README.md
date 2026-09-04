# Soparia PDV

Sistema de ponto de venda simples para a soparia. Roda no próprio computador
do estabelecimento, funciona offline no dia a dia, e não precisa de nenhuma
assinatura ou serviço pago.

## O que você precisa antes de começar

- O computador da soparia (Windows).
- Internet **só para o passo de instalação** (depois disso não precisa mais).
- [Node.js](https://nodejs.org) instalado, versão 22 ou mais nova — baixe a
  versão "LTS" no site e instale como qualquer programa (clicando em
  "Avançar" até o fim).

## Como instalar (fazer uma vez só)

1. Copie esta pasta do projeto para o computador da soparia.
2. Abra a pasta, clique com o botão direito dentro dela e escolha
   **"Abrir no Terminal"** (ou "Abrir janela do PowerShell aqui").
3. Digite o comando abaixo e aperte Enter (só precisa fazer isso uma vez):
   ```
   npm install
   ```
   Isso vai baixar tudo que o sistema precisa. Pode demorar alguns minutos.

## Como usar todo dia

1. Abra a pasta do projeto, botão direito → "Abrir no Terminal".
2. Digite:
   ```
   npm start
   ```
3. Você vai ver a mensagem `Soparia PDV rodando em http://localhost:3000`.
4. Abra o navegador (Chrome, Edge, etc.) e acesse:
   ```
   http://localhost:3000
   ```
5. Deixe essa janela do terminal aberta enquanto estiver usando o sistema —
   fechar ela desliga o programa.

Dica: se quiser, peça pra alguém criar um atalho na área de trabalho que
já abre o terminal e roda `npm start` automaticamente.

## Configurando a impressora da cozinha

O sistema imprime os pedidos automaticamente numa impressora térmica ligada
por USB. Para configurar:

1. No Windows, vá em **Configurações → Dispositivos → Impressoras e
   scanners** (ou "Dispositivos e Impressoras" no Painel de Controle) e
   anote o **nome exato** da impressora térmica (ex: `POS-58`).
2. Na pasta do projeto, copie o arquivo `printer.config.example.json`
   e renomeie a cópia para `printer.config.json`.
3. Abra `printer.config.json` num editor de texto (Bloco de Notas serve) e
   troque o texto pelo nome exato da sua impressora:
   ```json
   {
     "printerName": "POS-58"
   }
   ```
4. Salve o arquivo e reinicie o sistema (`Ctrl+C` no terminal, depois
   `npm start` de novo).

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
