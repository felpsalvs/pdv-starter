# Soparia PDV

Sistema de ponto de venda simples para a soparia. Roda no próprio computador
do estabelecimento, funciona offline no dia a dia, e não precisa de nenhuma
assinatura ou serviço pago.

## O que você precisa antes de começar

- O computador da soparia (Windows).
- Internet **só para o passo de instalação** (depois disso não precisa mais).
- [Node.js](https://nodejs.org) instalado — baixe a versão "LTS" no site e
  instale como qualquer programa (clicando em "Avançar" até o fim).

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
     "nomeImpressora": "POS-58"
   }
   ```
4. Salve o arquivo e reinicie o sistema (`Ctrl+C` no terminal, depois
   `npm start` de novo).

**Se a impressora estiver desligada ou não configurada, o sistema continua
funcionando normalmente** — o pedido é salvo e a tela avisa que não foi
possível imprimir, mas nada trava.

## Telas do sistema

- **Pedido**: tela principal para tirar os pedidos do dia a dia.
- **Cardápio**: cadastrar, editar preço e remover sopas do menu.
- **Caixa**: abrir o caixa no início do dia e fechar no final, conferindo o
  valor esperado com o valor contado na gaveta.

## Onde ficam os dados

Tudo é salvo num único arquivo em `data/pdv.db`. Para fazer backup, basta
copiar esse arquivo para um pendrive ou pasta na nuvem de tempos em tempos.
Não apague essa pasta `data`, ou você perde o histórico de pedidos e caixa.
