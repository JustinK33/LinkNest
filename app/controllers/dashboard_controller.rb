class DashboardController < ApplicationController
  def show
    @user = Current.session.user
    @links = @user.links.with_attached_resume_pdf.order(:position)

    # Generate QR code for user's public profile
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
end
