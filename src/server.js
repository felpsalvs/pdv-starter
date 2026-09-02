const express = require('express');
const path = require('path');

const sopasRouter = require('./routes/sopas');
const pedidosRouter = require('./routes/pedidos');
const caixaRouter = require('./routes/caixa');

const app = express();
const PORT = process.env.PORT || 3000;

app.use(express.json());
app.use(express.static(path.join(__dirname, '..', 'public')));

app.use('/api/sopas', sopasRouter);
app.use('/api/pedidos', pedidosRouter);
app.use('/api/caixa', caixaRouter);

app.listen(PORT, () => {
  console.log(`Soparia PDV rodando em http://localhost:${PORT}`);
});
