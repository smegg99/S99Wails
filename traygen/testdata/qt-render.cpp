// traygen/testdata/qt-render.cpp

#include <QCoreApplication>
#include <QImage>
#include <QPainter>
#include <QSvgRenderer>

int main(int argc, char **argv)
{
    QCoreApplication app(argc, argv);
    if (argc != 3) {
        return 2;
    }
    QSvgRenderer svg(QString::fromLocal8Bit(argv[1]));
    QImage image(32, 32, QImage::Format_ARGB32_Premultiplied);
    image.fill(Qt::transparent);
    QPainter painter(&image);
    svg.render(&painter);
    painter.end();
    return svg.isValid() && image.save(QString::fromLocal8Bit(argv[2])) ? 0 : 1;
}
