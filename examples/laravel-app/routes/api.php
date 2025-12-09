<?php

use Illuminate\Support\Facades\Route;

Route::get('/ping', function () {
    return response()->json(['message' => 'pong']);
});

Route::get('/health', function () {
    return response()->json(['status' => 'ok']);
});
