class HealthController < ApplicationController
  def ping
    render json: { message: 'pong' }
  end

  def status
    render json: { status: 'ok' }
  end
end
