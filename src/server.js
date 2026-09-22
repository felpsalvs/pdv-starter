const express = require('express');
const path = require('path');

const productsRouter = require('./routes/products');
const categoriesRouter = require('./routes/categories');
const ordersRouter = require('./routes/orders');
const cashRegisterRouter = require('./routes/cashRegister');
const reportRouter = require('./routes/report');
const { scheduleBackups } = require('./backup');

const app = express();
const PORT = process.env.PORT || 3000;

app.use(express.json());
app.use(express.static(path.join(__dirname, '..', 'dist')));

app.use('/api/products', productsRouter);
app.use('/api/categories', categoriesRouter);
app.use('/api/orders', ordersRouter);
app.use('/api/cash-register', cashRegisterRouter);
app.use('/api/report', reportRouter);

app.listen(PORT, () => {
  console.log(`Soparia PDV rodando em http://localhost:${PORT}`);
  scheduleBackups();
});
