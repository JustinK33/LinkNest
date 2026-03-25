class ProfilesController < ApplicationController
  allow_unauthenticated_access
  before_action :set_user, only: [:show, :qr_code]

  def show
    @links = @user.links.public_links.order(:position)

    # Create QR code for the public profile URL
    profile_url = user_profile_url(@user)
    qr_code = RQRCode::QRCode.new(profile_url)
    @qr_png = qr_code.as_png(
      resize_gte_to: false,
      resize_exactly_to: 300,
      fill: 'white',
      color: 'black',
      border_modules: 2
    )
  end

  def qr_code
    profile_url = user_profile_url(@user)
    qr_code = RQRCode::QRCode.new(profile_url)

    png = qr_code.as_png(
      resize_gte_to: false,
      resize_exactly_to: 400,
      fill: 'white',
      color: 'black',
      border_modules: 4
    )

    send_data png.to_s,
              type: 'image/png',
              disposition: 'inline',
              filename: "#{@user.slug}_qr_code.png"
  end

  private
    def set_user
      @user = User.find_by_slug!(params[:slug])
    rescue ActiveRecord::RecordNotFound
      redirect_to root_path, alert: "Profile not found"
    end
end
